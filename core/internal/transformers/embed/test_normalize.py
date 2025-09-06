import normalize
import pytest


def test_normalize():
    def test(d, expected):
        normalize._Norm.normalize(d)
        assert d == expected
        print(f"normalization of {repr(d)} is ok")

    def test_raises_err(d, match):
        with pytest.raises(ValueError, match=match):
            normalize._Norm.normalize(d)
        print(f"a ValueError has been correctly raised when normalizing {repr(d)}")

    # Test some valid normalizations.
    test(
        {},
        {},
    )
    test(
        {"a": 10},
        {"a": 10},
    )
    test(
        {"a": 10.42},
        {"a": 10.42},
    )
    test(
        {"a": [[42]]},
        {"a": [[42]]},
    )
    test(
        {"a": float("nan")},
        {"a": "NaN"},
    )
    test(
        {"a": [float("nan")]},
        {"a": ["NaN"]},
    )
    test(
        {"a": float("inf")},
        {"a": "Infinite"},
    )
    test(
        {"a": float("-inf")},
        {"a": "-Infinite"},
    )
    test(
        {"a": [[42]], "b": [[42]]},
        {"a": [[42]], "b": [[42]]},
    )
    test(
        {"a": [[float("nan")]], "b": [[42]]},
        {"a": [["NaN"]], "b": [[42]]},
    )
    test(
        {"a": [[True]], "b": [[42, "hello"]]},
        {"a": [[True]], "b": [[42, "hello"]]},
    )
    test(
        {"data": {"a": {"b": {"uuid": "550e8400-e29b-41d4-a716-446655440000"}}}},
        {"data": {"a": {"b": {"uuid": "550e8400-e29b-41d4-a716-446655440000"}}}},
    )

    # Test some normalizations that must fail.
    test_raises_err([], "transformed value is 'list', not 'dict'")
    test_raises_err(None, "transformed value is 'NoneType', not 'dict'")
    test_raises_err(42, "transformed value is 'int', not 'dict'")
    test_raises_err("some string", "transformed value is 'str', not 'dict'")

    # Test normalization on circular references.
    test_dict_ok = {"x": 20, "y": [20, 30]}
    test(
        test_dict_ok,
        test_dict_ok,
    )
    test(
        {"a": test_dict_ok, "b": test_dict_ok},
        {"a": test_dict_ok, "b": test_dict_ok},
    )
    test_dict_bad = {}
    test_dict_bad["x"] = test_dict_bad
    test_raises_err(test_dict_bad, "transformed value contains a circular reference")
