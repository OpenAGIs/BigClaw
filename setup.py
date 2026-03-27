"""Compatibility-only packaging shim for the frozen legacy Python tree."""

from setuptools import find_packages, setup


setup(
    name="bigclaw",
    version="0.1.0",
    description="BigClaw v1.0 engineering & ops execution platform (MVP)",
    package_dir={"": "src"},
    packages=find_packages("src"),
)
