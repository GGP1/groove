import React from "react";
import { StyleSheet } from "react-native";
import { Icon } from "react-native-elements";
import { TouchableOpacity } from "react-native-gesture-handler";

export const GoBackIcon = (color?: string) => (
	<TouchableOpacity style={styles.back} activeOpacity={0.6}>
		<Icon
			name="chevron-left"
			color={color}
			size={40}
		/>
	</TouchableOpacity>
);

const styles = StyleSheet.create({
	back: {
		backgroundColor: "rgba(255, 255, 255, 0.4)",
		borderRadius: 10,
	},
});
