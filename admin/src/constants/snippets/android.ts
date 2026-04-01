export const INSTALL_COMMAND =
	'Add the latest version of `com.krenalis.analytics.kotlin:android` to your `build.gradle`';

export const SNIPPET = `import com.krenalis.analytics.kotlin.android.Analytics

val client = Analytics("writekey", applicationContext) {
  endpoint = "endpoint"
  trackApplicationLifecycleEvents = true
  flushAt = 3
  flushInterval = 10
  // ...other config options
}`;

export const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/android-sdk';
