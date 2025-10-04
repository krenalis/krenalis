export const INSTALL_COMMAND = 'Install-Package Meergo.Analytics.CSharp';

export const SNIPPET = `using Meergo.Analytics;

var config = new Config()
    .SetEndpoint("endpoint");

Analytics.Initialize("writekey", config);`;

export const DOCUMENTATION_LINK = 'http://localhost:8080/connectors/sources/dotnet';
