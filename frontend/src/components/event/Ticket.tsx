import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Ticket } from "../../types/ticket";

interface Props {
	ticket: Ticket
}

/** Ticketx is the component used to showcase information about a ticket inside an event. */
export const Ticketx = (props: Props) => {
	const { ticket } = props;

	return (
		<View style={styles.container}>
			<Text>
				{ticket.name},
				{ticket.available_count},
				{ticket.cost},
				{ticket.linked_role},
			</Text>
		</View>
	);
};


const styles = StyleSheet.create({
	container: {

	},
});
