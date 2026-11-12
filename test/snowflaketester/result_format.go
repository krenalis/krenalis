// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

// Query tags identify the application reads observed by the transport overlay.
const (
	WarehouseValuesTag    = "krenalis-transport/warehouse-values"
	WarehouseReferenceTag = "krenalis-transport/warehouse-reference"
	WarehouseNativeTag    = "krenalis-transport/warehouse-native"
	ConnectorValuesTag    = "krenalis-transport/connector-values"
	ConnectorReferenceTag = "krenalis-transport/connector-reference"
	ConnectorNativeTag    = "krenalis-transport/connector-native"
)

const requestedResultFormat = ""

// RequestedResultFormat reports the format selected by a transport overlay,
// or an empty string in a normal build. It does not attest a server response.
func RequestedResultFormat() string {
	return requestedResultFormat
}
