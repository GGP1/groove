import { Alert, Linking, Platform, ToastAndroid } from "react-native";
import Geolocation from "react-native-geolocation-service";
import { Region } from "react-native-maps";
import { PERMISSIONS, request } from "react-native-permissions";
import { Cache, CacheKey } from "../utils/cache";

export const DEFAULT_REGION: Region = {
	latitude: 0,
	longitude: 0,
	latitudeDelta: 65,
	longitudeDelta: 65,
};

export const getUserLocation = async (callback: (region: Region) => void) => {
	const hasPermission = await hasLocationPermission();
	if (!hasPermission) {
		return;
	}

	Geolocation.getCurrentPosition(
		async (position) => {
			const currentRegion: Region = {
				latitude: position.coords.latitude,
				longitude: position.coords.longitude,
				latitudeDelta: 0.05,
				longitudeDelta: 0.05,
			};
			callback(currentRegion);
			Cache.set(CacheKey.UserLastRegion, currentRegion);
		},
		(err) => {
			const lastRegion = Cache.get<Region>(CacheKey.UserLastRegion);
			if (lastRegion) {
				callback(lastRegion);
			}
			console.log(err);
		},
		{
			accuracy: {
				android: "high",
				ios: "nearestTenMeters",
			},
			enableHighAccuracy: false,
			timeout: 10000,
			maximumAge: 10000,
			distanceFilter: 0,
			forceRequestLocation: true,
			showLocationDialog: false,
		},
	);
};

export const hasLocationPermission = async () => {
	if (Platform.OS === "ios") {
		const status = await request(PERMISSIONS.IOS.LOCATION_WHEN_IN_USE);
		switch (status) {
			case "granted":
				return true;
			case "denied":
				Alert.alert("Location permission denied");
				break;
			case "unavailable":
				Alert.alert(
					"Turn on Location Services to allow determine your location.",
					"",
					[
						{
							text: "Go to Settings", onPress: () => {
								Linking.openSettings().catch(() => {
									Alert.alert("Unable to open settings");
								});
							},
						},
						{ text: "Don't Use Location", onPress: () => { } },
					],
				);
				break;
			default:
				return false;
		}
	} else {
		if (Platform.Version < 23) {
			return true;
		}

		const status = await request(PERMISSIONS.ANDROID.ACCESS_FINE_LOCATION);
		switch (status) {
			case "granted":
				return true;
			case "denied":
				ToastAndroid.show("Location permission denied by user.", ToastAndroid.LONG);
				break;
			case "blocked":
				ToastAndroid.show("Location permission blocked by user.", ToastAndroid.LONG);
				break;
			default:
				return false;
		}
	}
};
