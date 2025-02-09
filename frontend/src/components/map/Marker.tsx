import { useNavigation } from "@react-navigation/native";
import React, { memo } from "react";
import { StyleSheet, Text, View } from "react-native";
import FastImage from "react-native-fast-image";
import { Callout, Marker as RNMarker } from "react-native-maps";
import { Event } from "../../types/event";
import { HomeNavigationProps } from "../../utils/navigation";
import { StrokeText } from "../StrokeText";

interface Props {
	event: Event
}

/** Marker is the component used for Google maps markers. */
export const Marker = memo((props: Props) => {
	const { event } = props;
	const navigation = useNavigation<HomeNavigationProps>();

	return (
		<RNMarker
			tracksViewChanges={false}
			coordinate={{
				latitude: event.location.coordinates.latitude,
				longitude: event.location.coordinates.longitude,
			}}
			opacity={0.9}
		>
			<View style={styles.markerContainer}>
				<FastImage source={{ uri: event.logo_url }} style={styles.pin} />
				<StrokeText containerStyle={styles.pinText} text={event.name} />
			</View>
			{/* TODO: on marker press, open a modal/view with the event's details instead of a callout */}
			<Callout
				onPress={() => navigation.navigate("Event", { event_id: event.id })}
				tooltip
				style={styles.callout}
			>
				<Text style={styles.title}>{event.name}</Text>
				<Text style={styles.description}>{event.description}</Text>
			</Callout>
		</RNMarker>
	);
});

const styles = StyleSheet.create({
	markerContainer: {
		justifyContent: "center",
		alignItems: "center",
		flexDirection: "row",
	},
	pin: {
		height: 40,
		width: 40,
	},
	pinText: {
		marginLeft: 5,
		paddingBottom: 3,
	},
	callout: {
		backgroundColor: "white",
	},
	title: {
		fontSize: 18,
		fontWeight: "bold",
	},
	description: {
		fontSize: 8,
		fontWeight: "normal",
	},
});
