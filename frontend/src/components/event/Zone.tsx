import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Zone } from "../../types/zone";

interface Props {
	zone: Zone,
}

/** Zonex .. */
export const Zonex = (props: Props) => {
	const { zone } = props;

	return (
		<View style={styles.container}>
			<Text>
				{zone.name},
				{zone.required_permission_keys}
			</Text>
		</View>
	);
};

const styles = StyleSheet.create({
	container: {

	},
});
