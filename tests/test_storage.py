"""Persistence contracts and production configuration failure behavior."""

from __future__ import annotations

import sys
import tempfile
import unittest
from datetime import date
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from gr import planner as PL  # noqa: E402
from gr import repo as R  # noqa: E402
from gr import session as SE  # noqa: E402
from gr import storage as ST  # noqa: E402


class MemoryStore:
    """Small contract fake proving sessions do not depend on repository writes."""

    def __init__(self):
        self.weeks: dict[date, str] = {}
        self.ticks: dict[date, set[str]] = {}
        self.events: list[dict] = []

    def load_week(self, sunday):
        return self.weeks.get(sunday)

    def previous_week(self, before):
        choices = [key for key in self.weeks if key < before]
        return self.weeks[max(choices)] if choices else ""

    def save_week(self, sunday, document):
        self.weeks[sunday] = document

    def read_ticks(self, sunday):
        return set(self.ticks.get(sunday, set()))

    def toggle_tick(self, sunday, key):
        rows = self.ticks.setdefault(sunday, set())
        if key in rows:
            rows.remove(key)
            return False
        rows.add(key)
        return True

    def record_decision(self, record):
        self.events.append(record)

    def state_ref(self, sunday):
        return f"memory:{sunday}"

    def ping(self):
        return None


class TestPersistenceBoundary(unittest.TestCase):
    def test_generated_week_survives_without_a_weeks_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ("items.md", "corpus.md", "candidates.md", "profile.md", "sides.md"):
                (root / name).write_text((ROOT / name).read_text(encoding="utf-8"),
                                         encoding="utf-8")
            (root / "recipes").symlink_to(ROOT / "recipes", target_is_directory=True)
            repo = R.load(root)
            result = PL.fill(repo, PL.PlannerResult(source="code"), nights=3)
            store = MemoryStore()
            sunday = date(2026, 8, 9)

            saved = SE.assemble(repo, result.meals, sunday, 3, 0, "code", store=store)
            reloaded = SE.load_existing(root, sunday, store=store)

            self.assertFalse((root / "weeks").exists())
            self.assertIsNotNone(reloaded)
            self.assertEqual([meal.slug for meal in reloaded.meals],
                             [meal.slug for meal in saved.meals])
            self.assertEqual(len(store.events), 1)
            self.assertEqual(saved.state_ref, "memory:2026-08-09")

    def test_ticks_are_durable_state_separate_from_the_rendered_plan(self):
        store = MemoryStore()
        sunday = date(2026, 8, 9)
        self.assertTrue(store.toggle_tick(sunday, "buy:onion"))
        self.assertEqual(store.read_ticks(sunday), {"buy:onion"})
        self.assertFalse(store.toggle_tick(sunday, "buy:onion"))
        self.assertEqual(store.read_ticks(sunday), set())


class TestProductionConfiguration(unittest.TestCase):
    def test_unknown_environment_name_cannot_accidentally_enable_file_storage(self):
        with self.assertRaisesRegex(ST.ConfigurationError, "APP_ENV"):
            ST.from_environment(ROOT, {"APP_ENV": "prod"})

    def test_production_requires_database_url(self):
        with self.assertRaisesRegex(ST.ConfigurationError, "DATABASE_URL"):
            ST.from_environment(ROOT, {"APP_ENV": "production"})

    def test_production_cannot_select_file_fallback(self):
        with self.assertRaisesRegex(ST.ConfigurationError, "development-only"):
            ST.from_environment(ROOT, {
                "APP_ENV": "production", "GROCERY_ROUTER_STORAGE": "file"
            })

    def test_production_requires_full_tls_verification(self):
        with self.assertRaisesRegex(ST.ConfigurationError, "sslmode=verify-full"):
            ST.from_environment(ROOT, {
                "APP_ENV": "production",
                "DATABASE_URL": "postgresql://user:secret@example.test:26257/defaultdb",
            })

    def test_cockroach_style_tls_url_is_accepted(self):
        store = ST.from_environment(ROOT, {
            "APP_ENV": "production",
            "DATABASE_URL": (
                "postgresql://user:secret@example.test:26257/defaultdb"
                "?sslmode=verify-full"
            ),
        })
        self.assertIsInstance(store, ST.DatabaseStore)

    def test_unavailable_database_raises_and_never_creates_file_state(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)

            def unavailable(*_args, **_kwargs):
                raise OSError("database unavailable")

            store = ST.DatabaseStore(
                "postgresql://user:secret@example.test/defaultdb?sslmode=verify-full",
                production=True,
                connect=unavailable,
            )
            with self.assertRaisesRegex(OSError, "database unavailable"):
                store.load_week(date(2026, 8, 9))
            self.assertEqual(list(root.iterdir()), [])


class FakeDatabaseConnection:
    """SQL-shape fake exercising DatabaseStore itself without external credentials."""

    def __init__(self):
        self.weeks = {}
        self.ticks = {}
        self.events = []

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def execute(self, sql, params=None):
        normalized = " ".join(sql.split())
        if normalized.startswith("INSERT INTO weekly_plans"):
            self.weeks[params[0]] = params[1]
            return FakeResult()
        if normalized.startswith("SELECT document FROM weekly_plans WHERE week_start ="):
            value = self.weeks.get(params[0])
            return FakeResult([(value,)] if value is not None else [])
        if normalized.startswith("SELECT document FROM weekly_plans WHERE week_start <"):
            choices = [day for day in self.weeks if day < params[0]]
            return FakeResult([(self.weeks[max(choices)],)] if choices else [])
        if normalized.startswith("SELECT item_key FROM shopping_ticks"):
            rows = [(key,) for key, checked in self.ticks.get(params[0], {}).items()
                    if checked]
            return FakeResult(rows)
        if normalized.startswith("INSERT INTO shopping_ticks"):
            rows = self.ticks.setdefault(params[0], {})
            rows[params[1]] = not rows.get(params[1], False)
            return FakeResult([(rows[params[1]],)])
        if normalized.startswith("INSERT INTO plan_events"):
            self.events.append(params)
            return FakeResult()
        if normalized.startswith("SELECT version FROM schema_migrations WHERE version ="):
            return FakeResult([(params[0],)])
        raise AssertionError(f"unexpected SQL: {normalized}")


class TestDatabaseStore(unittest.TestCase):
    def test_database_backend_round_trip_and_atomic_tick(self):
        connection = FakeDatabaseConnection()
        store = ST.DatabaseStore(
            "postgresql://local/defaultdb?sslmode=disable",
            connect=lambda *_args, **_kwargs: connection,
        )
        sunday = date(2026, 8, 9)
        store.save_week(sunday, "# durable plan\n")
        self.assertEqual(store.load_week(sunday), "# durable plan\n")
        self.assertEqual(store.previous_week(date(2026, 8, 10)), "# durable plan\n")
        self.assertTrue(store.toggle_tick(sunday, "buy:onion"))
        self.assertEqual(store.read_ticks(sunday), {"buy:onion"})
        self.assertFalse(store.toggle_tick(sunday, "buy:onion"))
        self.assertEqual(store.read_ticks(sunday), set())
        store.record_decision({"week": sunday.isoformat(), "kind": "proposed"})
        self.assertEqual(len(connection.events), 1)
        store.ping()

    def test_serialization_failure_retries_without_changing_backend(self):
        database = FakeDatabaseConnection()
        attempts = 0

        class SerializationFailure(RuntimeError):
            sqlstate = "40001"

        class ConflictingConnection:
            def __enter__(self):
                return self

            def __exit__(self, *_args):
                return False

            def execute(self, *_args, **_kwargs):
                raise SerializationFailure("restart transaction")

        def connect(*_args, **_kwargs):
            nonlocal attempts
            attempts += 1
            return ConflictingConnection() if attempts == 1 else database

        store = ST.DatabaseStore(
            "postgresql://local/defaultdb?sslmode=disable", connect=connect
        )
        sunday = date(2026, 8, 9)
        store.save_week(sunday, "# retried plan\n")
        self.assertEqual(attempts, 2)
        self.assertEqual(database.weeks[sunday], "# retried plan\n")


class FakeResult:
    def __init__(self, rows=()):
        self.rows = list(rows)

    def fetchall(self):
        return self.rows

    def fetchone(self):
        return self.rows[0] if self.rows else None


class FakeMigrationConnection:
    def __init__(self):
        self.applied = set()
        self.schema_scripts = []

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def execute(self, sql, params=None):
        normalized = " ".join(sql.split())
        if normalized.startswith("SELECT version FROM schema_migrations"):
            return FakeResult((name,) for name in sorted(self.applied))
        if normalized.startswith("INSERT INTO schema_migrations"):
            self.applied.add(params[0])
        elif (normalized.startswith("CREATE TABLE IF NOT EXISTS")
              or normalized.startswith("CREATE INDEX IF NOT EXISTS")):
            self.schema_scripts.append(sql)
        return FakeResult()


class TestMigrations(unittest.TestCase):
    def test_numbered_migration_is_explicit_and_idempotent(self):
        connection = FakeMigrationConnection()
        store = ST.DatabaseStore(
            "postgresql://local/defaultdb?sslmode=disable",
            connect=lambda *_args, **_kwargs: connection,
        )
        first = store.migrate(ROOT / "migrations")
        second = store.migrate(ROOT / "migrations")
        self.assertEqual(first, ["001_initial.sql"])
        self.assertEqual(second, [])
        schema = "\n".join(connection.schema_scripts)
        # The migration ledger's idempotent CREATE runs on each invocation; the four
        # statements in 001 run only once.
        self.assertEqual(len(connection.schema_scripts), 6)
        self.assertEqual(schema.count("CREATE TABLE IF NOT EXISTS weekly_plans"), 1)
        self.assertIn("shopping_ticks", schema)
        self.assertIn("plan_events", schema)


if __name__ == "__main__":
    unittest.main(verbosity=2)
