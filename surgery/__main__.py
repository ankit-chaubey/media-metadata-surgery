"""
Media Metadata Surgery - Python wrapper launcher.

This module locates the compiled Go binary bundled with the package
and delegates all arguments to it unchanged.
"""
import os
import subprocess
import sys
import platform


def _find_binary():
    """Locate the surgery binary, trying multiple candidate paths."""
    here = os.path.dirname(os.path.abspath(__file__))

    # Platform-specific binary names
    system = platform.system().lower()
    binary_name = "surgery"
    if system == "windows":
        binary_name = "surgery.exe"

    candidates = [
        os.path.join(here, "bin", binary_name),
        os.path.join(here, binary_name),
        os.path.join(os.path.dirname(here), binary_name),
    ]

    for path in candidates:
        if os.path.isfile(path) and os.access(path, os.X_OK):
            return path

    return None


def main():
    binary = _find_binary()
    if binary is None:
        print(
            "Error: surgery binary not found.\n"
            "If you installed via pip, the binary should be at:\n"
            f"  {os.path.dirname(os.path.abspath(__file__))}/bin/surgery\n\n"
            "To build from source:\n"
            "  git clone https://github.com/ankit-chaubey/media-metadata-surgery\n"
            "  cd media-metadata-surgery\n"
            "  go build -o surgery ./cli\n",
            file=sys.stderr,
        )
        sys.exit(1)

    result = subprocess.call([binary] + sys.argv[1:])
    sys.exit(result)


if __name__ == "__main__":
    main()
