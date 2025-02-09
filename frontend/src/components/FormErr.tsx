import React from "react";
import { StyleSheet, Text } from "react-native";

interface Props {
	cond?: boolean,
	err?: string
}

export const FormErr = (props: Props) => {
	if (props.cond && props.err) {
		return <Text style={styles.error}>{props.err}</Text>;
	}
	return null;
};

const styles = StyleSheet.create({
	error: {
		color: "crimson",
		fontWeight: "bold",
		fontSize: 12.5,
		textAlign: "center",
	},
});
