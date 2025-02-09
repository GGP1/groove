import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { Icon } from "react-native-elements";
import MapView, { PROVIDER_GOOGLE, Region } from "react-native-maps";
import { API } from "../../api/api";
import { Event } from "../../types/event";
import { DEFAULT_REGION, getUserLocation } from "../../utils/location";
import { Pressable } from "../Pressable";
import { Marker } from "./Marker";

/**
 * Once current region's latitude delta is higher than this value,
 * no requests will be made and no markers will be shown in the map
 */
const LAT_DELTA_THRESHOLD = 1.0;

/** Map contains Google Maps component and a location searcher */
export const Map = () => {
	const [currentRegion, setCurrentRegion] = useState<Region>(DEFAULT_REGION);
	const [events, setEvents] = useState<Event[]>();

	useEffect(() => {
		getUserLocation(region => {
			setCurrentRegion(region);
			searchEvents(region);
		});
	}, []);

	// TODO: rate limit search button pressing
	const searchEvents = async (region: Region) => {
		if (region.latitudeDelta > 1.2) {
			return;
		}
		try {
			const nearbyEvents = await API.Events.SearchLocation({
				latitude: region.latitude,
				longitude: region.longitude,
				latitude_delta: region.latitudeDelta,
				longitude_delta: region.longitudeDelta,
			});
			setEvents(nearbyEvents);
		} catch (err) {
			console.log(err);
		}
	};

	/* TODO: use geocoder service to translate an address into coordinates and create the top search bar.
	const searchLocation = async (input: string) => {
		const geocoder = (input: string): Region => {
			return { latitude: 1, longitude: 1, latitudeDelta: 1, longitudeDelta: 1 };
		};
		const coords = geocoder(input);
		setCurrentRegion(coords);
	};
	*/

	return (
		<View style={styles.container}>
			{/* TODO: let the user search for a location and use animateToRegion */}
			<MapView
				provider={PROVIDER_GOOGLE}
				style={styles.map}
				mapType="hybrid"
				loadingEnabled
				showsCompass={false}
				rotateEnabled={false}
				showsUserLocation
				followsUserLocation
				userLocationPriority="balanced"
				region={currentRegion}
				onRegionChangeComplete={(region) => setCurrentRegion(region)}
			>
				{currentRegion.latitudeDelta < LAT_DELTA_THRESHOLD && events?.map((event) => <Marker key={event.id} event={event} />)}
			</MapView>
			{currentRegion.latitudeDelta < LAT_DELTA_THRESHOLD && (
				<View style={styles.searchEvents}>
					<Pressable onPress={() => searchEvents(currentRegion)}>
						<Icon name="search" size={25} raised />
					</Pressable>
				</View>
			)}
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		...StyleSheet.absoluteFillObject,
		height: "100%",
		justifyContent: "flex-end",
		alignItems: "center",
	},
	map: {
		...StyleSheet.absoluteFillObject,
		marginTop: "5%",
	},
	overlay: {
		position: "absolute",
		bottom: 70,
	},
	searchEvents: {
		position: "absolute",
		bottom: 50,
		right: 20,
	},
});
