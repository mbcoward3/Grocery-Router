import pathlib
import tempfile
import unittest

from update_dev_image import update


class UpdateDevImageTest(unittest.TestCase):
    def test_digest_and_idempotence(self):
        digest = "sha256:" + "a" * 64
        old = "sha256:" + "b" * 64
        with tempfile.TemporaryDirectory() as directory:
            path = pathlib.Path(directory) / "release.yaml"
            original = (
                'spec:\n  values:\n    image:\n      digest: "' + old + '"\n'
                '    authSecret:\n      name: grocery-router-auth\n'
            )
            path.write_text(original)
            update(path, digest)
            expected = original.replace(old, digest)
            self.assertEqual(path.read_text(), expected)
            update(path, digest)
            self.assertEqual(path.read_text(), expected)

    def test_rejects_invalid_digest_missing_and_duplicate_image(self):
        image = '    image:\n      digest: "sha256:' + 'b' * 64 + '"\n'
        cases = [(image, "latest"), ("image: other:latest\n", "sha256:" + "a" * 64),
                 (image * 2, "sha256:" + "a" * 64)]
        for text, digest in cases:
            with self.subTest(text=text, digest=digest), tempfile.TemporaryDirectory() as directory:
                path = pathlib.Path(directory) / "release.yaml"
                path.write_text(text)
                with self.assertRaises(ValueError):
                    update(path, digest)
                self.assertEqual(path.read_text(), text)


if __name__ == "__main__":
    unittest.main()
