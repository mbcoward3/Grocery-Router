import pathlib
import tempfile
import unittest

from update_dev_image import update


class UpdateDevImageTest(unittest.TestCase):
    def test_tag_digest_and_idempotence(self):
        digest = "sha256:" + "a" * 64
        for old in (":sha-" + "b" * 40, "@sha256:" + "c" * 64):
            with self.subTest(old=old), tempfile.TemporaryDirectory() as directory:
                path = pathlib.Path(directory) / "deployment.yaml"
                original = (
                    "containers:\n  - name: grocery-router\n"
                    f"    image: ghcr.io/mbcoward3/grocery-router{old}\n"
                    "    envFrom:\n      - secretRef: {name: grocery-router-auth}\n"
                )
                path.write_text(original)
                update(path, digest)
                expected = original.replace(old, "@" + digest)
                self.assertEqual(path.read_text(), expected)
                update(path, digest)
                self.assertEqual(path.read_text(), expected)

    def test_rejects_invalid_digest_missing_and_duplicate_image(self):
        image = "image: ghcr.io/mbcoward3/grocery-router:latest\n"
        cases = [(image, "latest"), ("image: other:latest\n", "sha256:" + "a" * 64),
                 (image * 2, "sha256:" + "a" * 64)]
        for text, digest in cases:
            with self.subTest(text=text, digest=digest), tempfile.TemporaryDirectory() as directory:
                path = pathlib.Path(directory) / "deployment.yaml"
                path.write_text(text)
                with self.assertRaises(ValueError):
                    update(path, digest)
                self.assertEqual(path.read_text(), text)


if __name__ == "__main__":
    unittest.main()
