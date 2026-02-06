from setuptools import setup, find_packages

setup(
    name="surgery",
    version="0.1.1",
    packages=find_packages(),
    include_package_data=True,
    author="Ankit Chaubey",
    author_email="m.ankitchaubey@gmail.com",
    description="Precision-focused media metadata inspection tool built in Go and distributed via Python",
    long_description=open("README.md", encoding="utf-8").read(),
    long_description_content_type="text/markdown",
    url="https://github.com/ankit-chaubey/media-metadata-surgery",
    project_urls={
        "Author": "https://github.com/ankit-chaubey",
        "Source": "https://github.com/ankit-chaubey/media-metadata-surgery",
        "Issues": "https://github.com/ankit-chaubey/media-metadata-surgery/issues",
    },
    license="Apache-2.0",
    python_requires=">=3.7",
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: Apache Software License",
        "Operating System :: POSIX :: Linux",
        "Programming Language :: Python :: 3",
        "Programming Language :: Go",
        "Topic :: Multimedia :: Graphics",
        "Topic :: Security",
        "Topic :: Utilities",
    ],
    entry_points={
        "console_scripts": [
            "surgery=surgery.__main__:main",
        ]
    },
)
