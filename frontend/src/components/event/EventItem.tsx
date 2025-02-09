import { useNavigation } from "@react-navigation/core";
import React, { memo } from "react";
import { Dimensions, StyleSheet, Text, View } from "react-native";
import { Card, Icon } from "react-native-elements";
import FastImage from "react-native-fast-image";
import { Event, ticketTypeToString, typeToString } from "../../types/event";
import { Cron } from "../../utils/cron";
import { dayAndMonth } from "../../utils/date";
import { ExploreNavigationProps } from "../../utils/navigation";
import { Pressable } from "../Pressable";

interface Props {
	event: Event
}

/** EventItem is used to showcase a list of events. */
export const EventItem = memo((props: Props) => {
	const { event } = props;
	const navigation = useNavigation<ExploreNavigationProps>();
	const [cronExpr] = Cron.parse(event.start_date, event.end_date, event.cron);
	const nextDate = cronExpr.next().toDate();

	// TODO: would be cool to show an overlay above the image showing something like the logo (semi-transparent)
	return (
		<Pressable onPress={() => navigation.push("Event", { event_id: props.event.id })}>
			<Card containerStyle={styles.cardContainer}>
				<FastImage style={styles.image} source={{ uri: event.header_url && event.header_url }} />
				<View style={styles.details}>
					<Text style={styles.title}>{capitalize(event.name)}</Text>
					<Text style={styles.type}>{typeToString(event.type)}</Text>
					<View style={styles.row}>
						<Icon name="event" size={24} containerStyle={styles.icon} />
						<Text style={styles.content}>{dayAndMonth(nextDate)}</Text>
					</View>
					<View style={styles.row}>
						<Icon name="person" size={24} containerStyle={styles.icon} />
						<Text style={styles.content}>{event.slots} slots</Text>
					</View>
					<View style={styles.row}>
						<Icon name="attach-money" size={24} containerStyle={styles.icon} />
						<Text style={styles.content}>{ticketTypeToString(event.ticket_type)}</Text>
					</View>
				</View>
			</Card>
		</Pressable>
	);
});

const capitalize = (str: string): string => {
	const words = str.split(" ");

	for (let i = 0; i < words.length; i++) {
		words[i] = words[i][0].toUpperCase() + words[i].substr(1);
	}

	return words.join(" ");
};

const { width, height } = Dimensions.get("screen");

const styles = StyleSheet.create({
	cardContainer: {
		elevation: 1,
		borderRadius: 2,
		padding: 0,
		margin: 5,
		marginBottom: 5,
		width: width - (width / 12),
		height: height / 2.7,
	},
	image: {
		alignSelf: "flex-start",
		width: "100%",
		height: 140,
	},
	details: {
		marginVertical: 3,
		marginHorizontal: 10,
		height: 110,
	},
	title: {
		textAlign: "center",
		fontSize: 22,
		fontWeight: "bold",
		marginBottom: 3,
	},
	type: {
		fontSize: 19,
		fontWeight: "bold",
		color: "crimson",
	},
	content: {
		fontSize: 19,
	},
	row: {
		flexDirection: "row",
	},
	icon: {
		justifyContent: "center",
		marginRight: 5,
		marginBottom: 2,
	},
});
