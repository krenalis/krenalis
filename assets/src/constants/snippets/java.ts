export const INSTALL_COMMAND = 'Add the latest version of `com.meergo.analytics.java` to your `pom.xml`';

export const SNIPPET = `import com.meergo.analytics.Analytics;
import com.meergo.analytics.messages.TrackMessage;

final Analytics analytics =
    Analytics.builder("writekey")
        .endpoint("endpoint")
        .build();`;

export const DOCUMENTATION_LINK = 'http://localhost:8080/developers/java-sdk';
