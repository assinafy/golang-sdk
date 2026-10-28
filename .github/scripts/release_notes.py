"""Extract a stable version's notes from a Keep a Changelog document."""

import re
import sys


def release_notes(changelog, tag):
    """Return one nonempty version section, rejecting invalid or ambiguous input."""
    if not re.fullmatch(r"v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)", tag):
        raise ValueError("release tag must be vMAJOR.MINOR.PATCH")
    sections = re.split(r"^## ", changelog, flags=re.MULTILINE)
    heading = re.compile(r"\[" + re.escape(tag[1:]) + r"\] - \d{4}-\d{2}-\d{2}\n")
    matches = [section[heading.match(section).end():].strip()
               for section in sections[1:] if heading.match(section)]
    if len(matches) != 1 or not matches[0]:
        raise ValueError(f"CHANGELOG.md must contain one nonempty dated entry for {tag}")
    return matches[0] + "\n"


if __name__ == "__main__":
    try:
        if len(sys.argv) != 2:
            raise ValueError("usage: release_notes.py vMAJOR.MINOR.PATCH < CHANGELOG.md")
        print(release_notes(sys.stdin.read(), sys.argv[1]), end="")
    except ValueError as error:
        sys.exit(str(error))
