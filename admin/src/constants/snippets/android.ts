export const INSTALL_COMMAND = 'Add the latest version of `com.meergo.analytics.kotlin:android` to your `build.gradle`';

export const SNIPPET = `import com.meergo.analytics.kotlin.android.Analytics
import com.meergo.analytics.kotlin.core.*

val client = Analytics("writekey", applicationContext) {
  trackApplicationLifecycleEvents = true
  flushAt = 3
  flushInterval = 10
  // ...other config options
}`;

export const DOCUMENTATION_LINK = 'https://www.meergo.com/docs/integrations/android-sdk';
