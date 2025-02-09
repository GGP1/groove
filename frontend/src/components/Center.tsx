import React from "react";
import { StyleSheet } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

interface Props {
	children: React.ReactNode;
}

export const Center = (props: Props) => {
	return (
		<SafeAreaView style={styles.center} >
			{props.children}
		</SafeAreaView>
	);
};

const styles = StyleSheet.create({
	center: {
		flex: 1,
		alignItems: "center",
		justifyContent: "center",
	},
});
