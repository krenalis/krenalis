export const INSTALL_COMMAND = 'Add the latest version of `com.krenalis.analytics.java` to your `pom.xml`';

export const SNIPPET = `import com.krenalis.analytics.Analytics;
import com.krenalis.analytics.messages.TrackMessage;

final Analytics analytics =
    Analytics.builder("writekey")
        .endpoint("endpoint")
        .build();`;

export const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/java-sdk';
