export const INSTALL_COMMAND = 'Add the latest version of the SDK to your `Package.swift` or install it via Xcode';

export const SNIPPET = `import Meergo

let analytics = Analytics(configuration: Configuration(writeKey: "writekey")
    .endpoint("endpoint")
    .trackApplicationLifecycleEvents(true)
    .flushAt(3)
    .flushInterval(10)) // ...other config options`;

export const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/ios-sdk';
