"""Sample tests for security-base-python template."""

from importlib.metadata import version


def test_version():
    """Verify version can be imported."""
    v = version("security-base-python")
    assert v == "0.1.0"
