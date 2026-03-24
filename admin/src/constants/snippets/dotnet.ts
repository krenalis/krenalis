export const INSTALL_COMMAND = 'Install-Package Krenalis.Analytics.CSharp';

export const SNIPPET = `using Krenalis.Analytics;

var config = new Config()
    .SetEndpoint("endpoint");

Analytics.Initialize("writekey", config);`;

export const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/dotnet-sdk';
