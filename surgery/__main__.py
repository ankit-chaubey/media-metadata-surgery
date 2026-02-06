import os
import subprocess
import sys

def main():
    here = os.path.dirname(__file__)
    bin_path = os.path.join(here, "bin", "surgery")

    if not os.path.isfile(bin_path):
        print("Error: surgery binary not found.")
        sys.exit(1)

    subprocess.call([bin_path] + sys.argv[1:])

if __name__ == "__main__":
    main()
