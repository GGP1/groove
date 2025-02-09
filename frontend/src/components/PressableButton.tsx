import React from "react";
import { StyleSheet, Text } from "react-native";
import { TouchableOpacity } from "react-native-gesture-handler";

interface Props {
	onPress?: () => void,
	disabled?: boolean | undefined;
	children: React.ReactNode;
}

export const PressableButton = (props: Props) => {
	return (
		<TouchableOpacity
			disabled={props.disabled}
			onPress={props.onPress}
			style={styles.button}
		>
			<Text style={styles.text}>
				{props.children}
			</Text>
		</TouchableOpacity>
	);
};

const styles = StyleSheet.create({
	button: {
		flex: 1,
		alignSelf: "stretch",
		backgroundColor: "#FFF",
		borderRadius: 5,
		borderWidth: 1,
		borderColor: "#D34E66",
		marginHorizontal: 5,
	},
	text: {
		textAlign: "center",
		alignSelf: "center",
		color: "#D34E66",
		fontSize: 16,
		fontWeight: "600",
		paddingVertical: 10,
	},
});
