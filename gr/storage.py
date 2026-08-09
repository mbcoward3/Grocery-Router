"""Persistence boundary for generated weeks and shopping-list state.

The recipe catalogue and household profile remain reviewable markdown inputs. Generated
plans, list ticks, and decision events go through this module so production can use a
portable PostgreSQL connection instead of a container filesystem.
"""

from __future__ import annotations

import json
import os
import time
from datetime import date
from pathlib import Path
from typing import Callable, Protocol, TypeVar
from urllib.parse import parse_qs, urlparse

from . import weekfile as W


EXPECTED_SCHEMA_VERSION = "001_initial.sql"


class ConfigurationError(RuntimeError):
    """Runtime persistence configuration is absent or unsafe."""


class Store(Protocol):
    def load_week(self, sunday: date) -> str | None: ...
    def previous_week(self, before: date) -> str: ...
    def save_week(self, sunday: date, document: str) -> None: ...
    def read_ticks(self, sunday: date) -> set[str]: ...
    def toggle_tick(self, sunday: date, key: str) -> bool: ...
    def record_decision(self, record: dict) -> None: ...
    def state_ref(self, sunday: date) -> str: ...
    def ping(self) -> None: ...


class FileStore:
    """Transparent local-development backend preserving the original markdown workflow."""

    def __init__(self, root: Path | str):
        self.root = Path(root)

    def _path(self, sunday: date) -> Path:
        return W.week_path(self.root, sunday)

    def load_week(self, sunday: date) -> str | None:
        path = self._path(sunday)
        return path.read_text(encoding="utf-8") if path.exists() else None

    def previous_week(self, before: date) -> str:
        directory = self.root / "weeks"
        if not directory.exists():
            return ""
        files = sorted(p for p in directory.glob("*.md") if p.stem < before.isoformat())
        return files[-1].read_text(encoding="utf-8") if files else ""

    def save_week(self, sunday: date, document: str) -> None:
        W.write(self._path(sunday), document)

    def read_ticks(self, sunday: date) -> set[str]:
        return W.read_ticks(self._path(sunday))

    def toggle_tick(self, sunday: date, key: str) -> bool:
        return W.toggle_tick(self._path(sunday), key)

    def record_decision(self, record: dict) -> None:
        path = self.root / "decisions.jsonl"
        with path.open("a", encoding="utf-8") as fh:
            fh.write(json.dumps(record, ensure_ascii=False) + "\n")

    def state_ref(self, sunday: date) -> str:
        path = self._path(sunday)
        try:
            return str(path.relative_to(self.root))
        except ValueError:
            return str(path)

    def ping(self) -> None:
        # Input readability is what makes this backend ready. Writes are checked when used.
        if not self.root.is_dir():
            raise OSError(f"application root does not exist: {self.root}")


T = TypeVar("T")


class DatabaseStore:
    """CockroachDB/PostgreSQL implementation, with no filesystem fallback."""

    def __init__(self, database_url: str, *, production: bool = False,
                 connect: Callable | None = None, connect_timeout: int = 5):
        validate_database_url(database_url, production=production)
        self.database_url = database_url
        self.connect_timeout = connect_timeout
        self._connector = connect

    def _connect(self):
        connector = self._connector
        if connector is None:
            try:
                import psycopg
            except ImportError as exc:  # pragma: no cover - exercised in the container
                raise ConfigurationError(
                    "database storage requires the locked psycopg dependency"
                ) from exc
            connector = psycopg.connect
        return connector(self.database_url, connect_timeout=self.connect_timeout)

    def _write(self, operation: Callable[[object], T]) -> T:
        """Retry CockroachDB serializable conflicts, never switch storage backends."""
        for attempt in range(3):
            try:
                with self._connect() as connection:
                    return operation(connection)
            except Exception as exc:
                if getattr(exc, "sqlstate", None) != "40001" or attempt == 2:
                    raise
                time.sleep(0.05 * (2 ** attempt))
        raise AssertionError("unreachable")

    def load_week(self, sunday: date) -> str | None:
        with self._connect() as connection:
            row = connection.execute(
                "SELECT document FROM weekly_plans WHERE week_start = %s", (sunday,)
            ).fetchone()
        return row[0] if row else None

    def previous_week(self, before: date) -> str:
        with self._connect() as connection:
            row = connection.execute(
                """SELECT document FROM weekly_plans
                   WHERE week_start < %s ORDER BY week_start DESC LIMIT 1""",
                (before,),
            ).fetchone()
        return row[0] if row else ""

    def save_week(self, sunday: date, document: str) -> None:
        def save(connection):
            connection.execute(
                """INSERT INTO weekly_plans (week_start, document)
                   VALUES (%s, %s)
                   ON CONFLICT (week_start) DO UPDATE
                   SET document = excluded.document, updated_at = now()""",
                (sunday, document),
            )
        self._write(save)

    def read_ticks(self, sunday: date) -> set[str]:
        with self._connect() as connection:
            rows = connection.execute(
                """SELECT item_key FROM shopping_ticks
                   WHERE week_start = %s AND checked = true""",
                (sunday,),
            ).fetchall()
        return {row[0] for row in rows}

    def toggle_tick(self, sunday: date, key: str) -> bool:
        def toggle(connection):
            row = connection.execute(
                """INSERT INTO shopping_ticks (week_start, item_key, checked)
                   VALUES (%s, %s, true)
                   ON CONFLICT (week_start, item_key) DO UPDATE
                   SET checked = NOT shopping_ticks.checked, updated_at = now()
                   RETURNING checked""",
                (sunday, key),
            ).fetchone()
            return bool(row[0])
        return self._write(toggle)

    def record_decision(self, record: dict) -> None:
        payload = json.dumps(record, ensure_ascii=False, separators=(",", ":"))

        def save(connection):
            connection.execute(
                "INSERT INTO plan_events (week_start, event) VALUES (%s, %s::JSONB)",
                (date.fromisoformat(record["week"]), payload),
            )
        self._write(save)

    def state_ref(self, sunday: date) -> str:
        return f"database week {sunday.isoformat()}"

    def ping(self) -> None:
        with self._connect() as connection:
            row = connection.execute(
                "SELECT version FROM schema_migrations WHERE version = %s",
                (EXPECTED_SCHEMA_VERSION,),
            ).fetchone()
        if not row:
            raise ConfigurationError(
                f"database schema is missing {EXPECTED_SCHEMA_VERSION}"
            )

    def migrate(self, migrations_dir: Path | str) -> list[str]:
        directory = Path(migrations_dir)
        files = sorted(directory.glob("[0-9][0-9][0-9]_*.sql"))
        if not files:
            raise ConfigurationError(f"no migrations found in {directory}")

        def apply(connection):
            connection.execute(
                """CREATE TABLE IF NOT EXISTS schema_migrations (
                       version TEXT PRIMARY KEY,
                       applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
                   )"""
            )
            applied = {
                row[0] for row in connection.execute(
                    "SELECT version FROM schema_migrations"
                ).fetchall()
            }
            changed = []
            for path in files:
                if path.name in applied:
                    continue
                statements = [
                    statement.strip()
                    for statement in path.read_text(encoding="utf-8").split(";")
                    if statement.strip()
                ]
                for statement in statements:
                    connection.execute(statement)
                connection.execute(
                    "INSERT INTO schema_migrations (version) VALUES (%s)", (path.name,)
                )
                changed.append(path.name)
            return changed

        return self._write(apply)


def validate_database_url(database_url: str, *, production: bool) -> None:
    parsed = urlparse(database_url)
    if parsed.scheme not in {"postgres", "postgresql"} or not parsed.hostname:
        raise ConfigurationError("DATABASE_URL must be a PostgreSQL connection URL")
    sslmode = parse_qs(parsed.query).get("sslmode", [""])[-1].lower()
    if production and sslmode != "verify-full":
        raise ConfigurationError(
            "production DATABASE_URL must include sslmode=verify-full"
        )


def from_environment(root: Path | str = ".", environ: dict[str, str] | None = None) -> Store:
    """Select storage explicitly; production can never drift to local files."""
    env = os.environ if environ is None else environ
    app_env = env.get("APP_ENV", "development").strip().lower()
    if app_env not in {"development", "test", "production"}:
        raise ConfigurationError(
            "APP_ENV must be 'development', 'test', or 'production'"
        )
    production = app_env == "production"
    database_url = env.get("DATABASE_URL", "").strip()
    database_url_file = env.get("DATABASE_URL_FILE", "").strip()
    if database_url and database_url_file:
        raise ConfigurationError(
            "set only one of DATABASE_URL or DATABASE_URL_FILE"
        )
    if database_url_file:
        try:
            database_url = Path(database_url_file).read_text(encoding="utf-8").strip()
        except OSError as exc:
            raise ConfigurationError("DATABASE_URL_FILE cannot be read") from exc
        if not database_url:
            raise ConfigurationError("DATABASE_URL_FILE is empty")
    backend = env.get("GROCERY_ROUTER_STORAGE", "").strip().lower()
    if not backend:
        backend = "database" if database_url or production else "file"

    if backend == "database":
        if not database_url:
            raise ConfigurationError(
                "database storage selected but DATABASE_URL is not set"
            )
        return DatabaseStore(database_url, production=production)
    if backend == "file":
        if production:
            raise ConfigurationError(
                "file storage is development-only; production requires DATABASE_URL"
            )
        # Containers can keep mutable development state on a dedicated volume while
        # loading catalogue inputs from the immutable application root. Direct local
        # runs retain the original repository-backed behavior when this is unset.
        file_root = env.get("GROCERY_ROUTER_FILE_ROOT", "").strip()
        return FileStore(file_root or root)
    raise ConfigurationError(
        "GROCERY_ROUTER_STORAGE must be either 'database' or 'file'"
    )
