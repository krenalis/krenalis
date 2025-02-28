#
# SPDX-License-Identifier: Elastic-2.0
#
#
# Copyright (c) 2025 Open2b
#

from datetime import date

import pandas as pd

pd.DataFrame(
    [
        {"parquet_id": 100, "first_name": "John", "last_name": "Lemon"},
        {"parquet_id": 101, "first_name": "Ringo", "last_name": "Planett"},
        {"parquet_id": 102, "phone_numbers": ["111 222", "333 444"]},
        {"parquet_id": 103, "address": {"street": "ABC1", "zip_code": 1234}},
        {"parquet_id": 104, "date_of_birth": date(1980, 1, 2)},  # after 1970-01-01
        {"parquet_id": 105, "date_of_birth": date(1935, 1, 2)},  # before 1970-01-01
    ]
).to_parquet("test.parquet")
