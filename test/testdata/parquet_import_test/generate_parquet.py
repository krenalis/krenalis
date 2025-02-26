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
    ]
).to_parquet("test.parquet")
