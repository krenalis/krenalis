#
# SPDX-License-Identifier: Elastic-2.0
#
#
# Copyright (c) 2025 Open2b
#

import pandas as pd

pd.DataFrame(
    [
        {"parquet_id": 100, "first_name": "John", "last_name": "Lemon"},
        {"parquet_id": 101, "first_name": "Ringo", "last_name": "Planett"},
        {"parquet_id": 102, "phone_numbers": ["111 222", "333 444"]},
        {"parquet_id": 103, "address": {"street": "ABC1", "zip_code": 1234}},
    ]
).to_parquet("test.parquet")
