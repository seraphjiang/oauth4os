from setuptools import setup, find_packages

setup(
    name="oauth4os",
    version="0.1.0",
    description="Python SDK for oauth4os — OAuth 2.0 proxy for OpenSearch",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    author="oauth4os contributors",
    url="https://github.com/seraphjiang/oauth4os",
    packages=find_packages(),
    python_requires=">=3.8",
    install_requires=["requests>=2.25.0"],
    classifiers=[
        "License :: OSI Approved :: Apache Software License",
        "Programming Language :: Python :: 3",
    ],
)
