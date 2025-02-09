import React from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "react-native-elements";

interface Props {
	k: string,
	v: string | number | undefined,
}

export const KV = (props: Props) => {
	return (
		<View style={styles.container}>
			<Text style={styles.key}>{props.k}: </Text>
			<Text style={styles.value}>{props.v}</Text>
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		flexDirection: "row",
	},
	key: {
		fontWeight: "bold",
	},
	value: {
		fontSize: 14,
	},
});
