import json

params = json.loads(
    """{"firstname":"Brian"}
"""
)


def f(params):
    return params["firstname"]


print(json.dumps(f(params)))
