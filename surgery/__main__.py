"""
Media Metadata Surgery - Python wrapper launcher.

Locates the correct compiled Go binary for the current platform/arch
and delegates all arguments to it unchanged.
"""
import os
import subprocess
import sys
import platform


def _binary_name():
    """Return the platform+arch specific binary filename."""
    system = platform.system().lower()   # linux, darwin, windows
    machine = platform.machine().lower() # x86_64, aarch64, arm64...

    # Normalise arch
    if machine in ("x86_64", "amd64"):
        arch = "amd64"
    elif machine in ("aarch64", "arm64"):
        arch = "arm64"
    else:
        arch = machine  # best-effort fallback

    if system == "windows":
        return f"surgery-windows-{arch}.exe"
    elif system == "darwin":
        return f"surgery-darwin-{arch}"
    else:
        # Linux, Android/Termux, and everything else
        return f"surgery-linux-{arch}"


def _find_binary():
    """
    Search for the surgery binary in candidate locations.

    Search order:
      1. <package_dir>/bin/<platform-binary>   (pip install)
      2. <package_dir>/bin/surgery             (generic fallback)
      3. Flat layout fallbacks
    """
    here = os.path.dirname(os.path.abspath(__file__))
    name = _binary_name()

    candidates = [
        os.path.join(here, "bin", name),           # platform-specific ✓
        os.path.join(here, "bin", "surgery"),      # generic linux fallback
        os.path.join(here, "bin", "surgery.exe"),  # generic windows fallback
        os.path.join(here, name),
        os.path.join(here, "surgery"),
    ]

    for path in candidates:
        if os.path.isfile(path) and os.access(path, os.X_OK):
            return path, None

    return None, name  # failure: return expected name for error message


def main():
    binary, expected_name = _find_binary()

    if binary is None:
        here = os.path.dirname(os.path.abspath(__file__))
        bin_dir = os.path.join(here, "bin")

        print(
            f"Error: surgery binary not found for your platform.\n"
            f"Expected: {expected_name}\n"
            f"Looked in: {bin_dir}\n",
            file=sys.stderr,
        )
        print("Available files in bin/:", file=sys.stderr)
        if os.path.isdir(bin_dir):
            for f in sorted(os.listdir(bin_dir)):
                fpath = os.path.join(bin_dir, f)
                ok = os.access(fpath, os.X_OK)
                print(f"  {'✓' if ok else '✗'} {f}", file=sys.stderr)
        else:
            print("  (bin/ directory missing)", file=sys.stderr)

        print(
            f"\nTo build from source:\n"
            f"  git clone https://github.com/ankit-chaubey/media-metadata-surgery\n"
            f"  cd media-metadata-surgery\n"
            f"  go build -o surgery/bin/{expected_name} ./cli\n",
            file=sys.stderr,
        )
        sys.exit(1)

    sys.exit(subprocess.call([binary] + sys.argv[1:]))


if __name__ == "__main__":
    main()
