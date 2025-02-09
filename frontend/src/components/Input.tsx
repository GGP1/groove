import React from "react";
import { Dimensions, StyleProp, StyleSheet, TextInput, TextInputProps, TextStyle, View, ViewStyle } from "react-native";

const { width } = Dimensions.get("screen");

interface Props extends TextInputProps {
	containerStyle?: StyleProp<ViewStyle> | undefined;
	inputStyle?: StyleProp<TextStyle> | undefined;
}

export const Input = (props: Props) => {
	return (
		<View style={[styles.container, props.containerStyle]}>
			<TextInput
				style={[styles.input, props.inputStyle]}
				{...props}
			/>
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		width: width / 1.15,
		alignSelf: "center",
		backgroundColor: "#e3e3e3",
		borderRadius: 5,
		marginVertical: 7,
	},
	input: {
		padding: 10,
	},
});
