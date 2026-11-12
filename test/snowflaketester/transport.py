# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Closed query tags used by numeric transport fixtures and their overlay."""

RESULT_TAGS = tuple("krenalis-transport/" + path + "-" + role
                    for path in ("warehouse", "connector")
                    for role in ("values", "reference", "native"))
