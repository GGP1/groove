import React from "react";
import { StyleProp, StyleSheet, Text, View, ViewStyle } from "react-native";

interface Props {
	containerStyle?: StyleProp<ViewStyle>,
	text: string
}

export const StrokeText = (props: Props) => {
	return (
		<View style={props.containerStyle}>
			<Text style={styles.paragraph}>{props.text}</Text>
			<Text style={[styles.paragraph, styles.abs, { textShadowOffset: { width: -2, height: -2 } }]}>{props.text}</Text>
			<Text style={[styles.paragraph, styles.abs, { textShadowOffset: { width: -2, height: 2 } }]}>{props.text}</Text>
			<Text style={[styles.paragraph, styles.abs, { textShadowOffset: { width: 2, height: -2 } }]}>{props.text}</Text>
		</View>
	);
};

const styles = StyleSheet.create({
	paragraph: {
		fontSize: 18,
		color: "#FFF",
		textShadowColor: "black",
		textShadowRadius: 1,
		textShadowOffset: {
			width: 2,
			height: 2,
		},
	},
	abs: {
		position: "absolute",
		top: 0,
		right: 0,
		bottom: 0,
		left: 0,
	},
});
