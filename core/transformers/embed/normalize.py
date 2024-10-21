class _Norm:
    @staticmethod
    def _infinite_to_string(n):
        import math

        if math.isnan(n):
            return "NaN"
        if n > 0:
            return "Infinite"
        return "-Infinite"

    @staticmethod
    def normalize(obj: dict) -> None:
        import math

        if type(obj) is not dict:
            raise ValueError(f"transformed value is '{type(obj).__name__}', not 'dict'")

        def norm(obj: dict, s: set) -> None:
            if id(obj) in s:
                raise ValueError("transformed value contains a circular reference")
            s.add(id(obj))
            if type(obj) is dict:
                for k, v in obj.items():
                    if type(v) in [dict, list]:
                        norm(v, s)
                    elif type(v) is float and not math.isfinite(v):
                        obj[k] = _Norm._infinite_to_string(v)
            elif type(obj) is list:
                for i, v in enumerate(obj):
                    if type(v) in [dict, list]:
                        norm(v, s)
                    elif type(v) is float and not math.isfinite(v):
                        obj[i] = _Norm._infinite_to_string(v)
            s.remove(id(obj))

        norm(obj, set())
