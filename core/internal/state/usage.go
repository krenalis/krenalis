// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

// organizationUsage tracks resource usage and limits for an organization.
type organizationUsage struct {
	limits         OrganizationLimits
	counts         OrganizationCounts
	connectorUsage map[*Connector]int // usage count per connector (connections + pipelines)
}

// newOrganizationUsage returns usage state with the given limits.
func newOrganizationUsage(limits OrganizationLimits) organizationUsage {
	return organizationUsage{
		limits:         limits,
		connectorUsage: map[*Connector]int{},
	}
}

// addAccessKey records an access key in the usage state.
func (usage *organizationUsage) addAccessKey() {
	usage.counts.AccessKeys++
}

// addConnection records a connection in the usage state.
func (usage *organizationUsage) addConnection(connector *Connector) {
	usage.addConnector(connector)
	usage.counts.Connections++
}

// addConnector adds one connector use.
func (usage *organizationUsage) addConnector(connector *Connector) {
	if count, ok := usage.connectorUsage[connector]; ok {
		usage.connectorUsage[connector] = count + 1
		return
	}
	usage.connectorUsage[connector] = 1
	usage.counts.Connectors++
}

// addMember records a member in the usage state.
func (usage *organizationUsage) addMember() {
	usage.counts.Members++
}

// addPipeline records a pipeline in the usage state.
func (usage *organizationUsage) addPipeline(format *Connector) {
	if format != nil {
		usage.addConnector(format)
	}
	usage.counts.Pipelines++
}

// addWorkspace records a workspace in the usage state.
func (usage *organizationUsage) addWorkspace() {
	usage.counts.Workspaces++
}

// currentCounts returns the resource counts.
func (usage *organizationUsage) currentCounts() OrganizationCounts {
	return usage.counts
}

// currentLimits returns the resource limits.
func (usage *organizationUsage) currentLimits() OrganizationLimits {
	return usage.limits
}

// isAccessKeyLimitReached returns whether the access key limit is reached.
func (usage *organizationUsage) isAccessKeyLimitReached() (bool, int) {
	return usage.counts.AccessKeys >= usage.limits.AccessKeys, usage.limits.AccessKeys
}

// isConnectionLimitReached returns whether the connection limit is reached.
func (usage *organizationUsage) isConnectionLimitReached() (bool, int) {
	return usage.counts.Connections >= usage.limits.Connections, usage.limits.Connections
}

// isConnectorLimitReached returns whether the connector limit is reached.
func (usage *organizationUsage) isConnectorLimitReached() (bool, int) {
	return usage.counts.Connectors >= usage.limits.Connectors, usage.limits.Connectors
}

// isConnectorUsed returns whether the connector is currently used.
func (usage *organizationUsage) isConnectorUsed(connector *Connector) bool {
	_, used := usage.connectorUsage[connector]
	return used
}

// isMemberLimitReached returns whether the member limit is reached.
func (usage *organizationUsage) isMemberLimitReached() (bool, int) {
	return usage.counts.Members >= usage.limits.Members, usage.limits.Members
}

// isPipelineLimitReached returns whether the pipeline limit is reached.
func (usage *organizationUsage) isPipelineLimitReached() (bool, int) {
	return usage.counts.Pipelines >= usage.limits.Pipelines, usage.limits.Pipelines
}

// isWorkspaceLimitReached returns whether the workspace limit is reached.
func (usage *organizationUsage) isWorkspaceLimitReached() (bool, int) {
	return usage.counts.Workspaces >= usage.limits.Workspaces, usage.limits.Workspaces
}

// removeAccessKey removes an access key from the usage state.
func (usage *organizationUsage) removeAccessKey() {
	usage.counts.AccessKeys--
}

// removeConnection removes a connection and its pipelines from the usage state.
func (usage *organizationUsage) removeConnection(connection *Connection) {
	usage.removeConnector(connection.connector)
	usage.counts.Connections--
	for _, pipeline := range connection.pipelines {
		usage.removePipeline(pipeline.format)
	}
}

// removeConnector removes one connector use.
func (usage *organizationUsage) removeConnector(connector *Connector) {
	if count := usage.connectorUsage[connector]; count > 1 {
		usage.connectorUsage[connector] = count - 1
		return
	}
	delete(usage.connectorUsage, connector)
	usage.counts.Connectors--
}

// removeMember removes a member from the usage state.
func (usage *organizationUsage) removeMember() {
	usage.counts.Members--
}

// removePipeline removes a pipeline from the usage state.
func (usage *organizationUsage) removePipeline(format *Connector) {
	if format != nil {
		usage.removeConnector(format)
	}
	usage.counts.Pipelines--
}

// removeWorkspace removes a workspace and its connections from the usage state.
func (usage *organizationUsage) removeWorkspace(workspace *Workspace) {
	usage.counts.Workspaces--
	for _, connection := range workspace.connections {
		usage.removeConnection(connection)
	}
}

// setLimits updates the resource limits.
func (usage *organizationUsage) setLimits(limits OrganizationLimits) {
	usage.limits = limits
}

// updatePipelineFormat updates connector usage for a pipeline format change.
func (usage *organizationUsage) updatePipelineFormat(oldFormat, newFormat *Connector) {
	if oldFormat == newFormat {
		return
	}
	if oldFormat != nil {
		usage.removeConnector(oldFormat)
	}
	if newFormat != nil {
		usage.addConnector(newFormat)
	}
}
