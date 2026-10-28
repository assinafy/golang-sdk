"""Checks for version-specific GitHub Release notes."""

import unittest

from release_notes import release_notes


class ReleaseNotesTests(unittest.TestCase):
    def test_selects_exact_version_and_preserves_markdown(self):
        changelog = """# Changelog

## [2.10.0] - 2026-12-01

### Added

- Another version.

## [2.1.0] - 2026-10-28

### Added

- Contract tracking.

### Security

- Credential protection.

## [2.0.1] - 2026-10-19

- Certificate validation.
"""
        self.assertEqual(
            release_notes(changelog, "v2.1.0"),
            "### Added\n\n- Contract tracking.\n\n### Security\n\n- Credential protection.\n",
        )
        self.assertEqual(release_notes(changelog, "v2.0.1"), "- Certificate validation.\n")

    def test_rejects_missing_empty_duplicate_or_undated_entries(self):
        entry = "## [2.1.0] - 2026-10-28\n\n- Contract tracking.\n"
        for text in ("", "## [2.1.0] - 2026-10-28\n\n", entry + entry, "## [2.1.0]\n\n- Notes.\n"):
            with self.subTest(text=text), self.assertRaises(ValueError):
                release_notes(text, "v2.1.0")

    def test_rejects_nonrelease_refs(self):
        for tag in ("main", "2.1.0", "v2.1", "v2.01.0", "v2.1.0-rc.1", "v2.1.0\n", "v2.1.0; echo unsafe"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                release_notes("## [2.1.0] - 2026-10-28\n\n- Notes.\n", tag)


if __name__ == "__main__":
    unittest.main()
