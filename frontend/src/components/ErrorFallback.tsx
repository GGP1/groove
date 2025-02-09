import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Button } from "react-native-elements";

interface Props {
	error: Error,
	resetErrorBoundary: (...args: unknown[]) => void;
}

export const ErrorFallback = (props: Props) => {
	return (
		<View style={styles.container}>
			<Text style={styles.title}>Something went wrong</Text>
			<Text style={styles.error}>{props.error}</Text>
			<Button title="Try again" onPress={() => props.resetErrorBoundary()} />
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		flex: 1,
		alignItems: "center",
		justifyContent: "center",
	},
	title: {
		fontSize: 30,
	},
	error: {
		fontSize: 20,
		marginVertical: 20,
	},
});
