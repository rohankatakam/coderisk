from setuptools import setup, find_packages

setup(
    name="crisk",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        # "cognee>=0.1.0",  # Temporarily disabled for MVP testing
        "click>=8.0.0",
        "gitpython>=3.1.0",
        "rich>=13.0.0",
        "pydantic>=2.0.0",
        # "tree-sitter>=0.20.0",  # Temporarily disabled for MVP
        # "tree-sitter-python>=0.20.0",
        # "tree-sitter-javascript>=0.20.0",
        # "tree-sitter-typescript>=0.20.0",
    ],
    entry_points={
        "console_scripts": [
            "crisk=crisk.cli:main",
        ],
    },
    author="CodeRisk Team",
    author_email="team@coderisk.dev",
    description="AI-powered code regression risk assessment",
    long_description="CodeRisk predicts code regression risk before you commit, with a focus on migration safety and team scaling scenarios.",
    url="https://github.com/coderisk/crisk",
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
    ],
    python_requires=">=3.8",
)