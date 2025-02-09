import React from "react";
import { StyleProp, TouchableOpacity, ViewStyle } from "react-native";

interface Props {
	onPress?: () => void,
	style?: StyleProp<ViewStyle>,
	children: any
}

export const Pressable = (props: Props) => {
	return (
		<TouchableOpacity
			onPress={props.onPress}
			style={props.style}
			activeOpacity={0.6}
		>
			{props.children}
		</TouchableOpacity>
	);
};
