import messaging, { firebase } from "@react-native-firebase/messaging";
import { HomeNavigationProps } from "./navigation";

// https://github.com/invertase/react-native-firebase/tree/master/docs/messaging
// https://github.com/invertase/react-native-firebase/blob/master/docs/messaging/notifications.md


class Firebase {

	constructor() {
		// @ts-ignore
		!firebase.apps.length ? firebase.initializeApp({}) : firebase.app();
	}

	async getToken(): Promise<string | null> {
		const ok = await this.requestPermission();
		if (!ok) {
			return null;
		}
		// messaging().getAPNSToken(), we are using firebase only for now
		const token = await messaging().getToken();
		return token;
	}

	deleteToken() {
		messaging().deleteToken();
	}

	async requestPermission(): Promise<boolean> {
		if (!messaging().hasPermission()) {
			const authStatus = await messaging().requestPermission();
			if (authStatus === messaging.AuthorizationStatus.AUTHORIZED ||
				authStatus === messaging.AuthorizationStatus.PROVISIONAL) {
				return true;
			} else {
				return false;
			}
		}
		return true;
	}

	/**
	 * onMessageForeground handles the notifications received while the app is in the **foreground**.
	 */
	onMessageForeground(callback: () => void) {
		messaging().onMessage(async remoteMessage => {
			const notification = JSON.stringify(remoteMessage);
			console.log("Foreground notification", notification);
			callback();
		});
	}

	/**
	 * onMessageBackground handles the notifications received while the app is in **background** or **quit**.
	 *
	 * This method must be called **outside** of the application lifecycle, e.g. alongside
	 * `AppRegistry.registerComponent()` method call at the the entry point of the application code.
	 */
	onMessageBackground() {
		// If the `RemoteMessage` payload contains a notification property,
		// the device will have displayed a notification to the user (handle data only).
		messaging().setBackgroundMessageHandler(async remoteMessage => {
			const notification = JSON.stringify(remoteMessage);
			console.log("Background notification", notification);
		});
	}

	/** Handle notifications when the application is running, but in the background
	 * @param navigation navigate to a desired screen
	 * TODO: opens the app but does not recharges it and it executes the console log multiple times
	*/
	onNotificationOpenedApp(navigation: HomeNavigationProps) {
		messaging().onNotificationOpenedApp(remoteMessage => {
			console.log(
				"Notification caused app to open from background state:",
				remoteMessage.notification,
			);
			// TODO: gather info form the notification metadata and navigate depending on that
			navigation.navigate("Home");
		});
	}

	/** Handle notifications when the application is opened from a quit state.
	 * @param fn function to execute when the remote message is received.
	 * message should be of type RemoteMessage but the import is broken.
	*/
	getInitialNotification(fn: (message: any) => void) {
		messaging().getInitialNotification().then(remoteMessage => {
			if (remoteMessage) {
				console.log(
					"Notification caused app to open from quit state:",
					remoteMessage.notification,
				);

				fn(remoteMessage);
			}
		});
	}
}

export const Notifications: Firebase = new Firebase();
