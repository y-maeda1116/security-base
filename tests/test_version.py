"""Sample tests for security-base-python template."""


def test_version():
    """Verify version can be imported."""
    from importlib.metadata import version

    v = version("security-base-python")
    assert v == "0.1.0"


def test_placeholder():
    """Placeholder test to verify test infrastructure works."""
    assert 1 + 1 == 2
