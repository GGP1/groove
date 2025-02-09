import React, { useState } from "react";
import { Dimensions, NativeSyntheticEvent, StyleSheet, TextInput, TextInputFocusEventData, View } from "react-native";
import { Icon } from "react-native-elements";

const { width } = Dimensions.get("screen");

interface Props {
	onChangeText: (text: string) => void;
	onBlur?: (e: NativeSyntheticEvent<TextInputFocusEventData>) => void;
	value?: string,
	placeholder?: string
}

export const PasswordInput = (props: Props) => {
	const [iconName, setIconName] = useState<string>("visibility-off");
	const [showPassword, setShowPassword] = useState<boolean>(false);

	return (
		<View style={styles.container}>
			<TextInput
				autoCompleteType={"password"}
				autoCapitalize={"none"}
				style={styles.input}
				placeholder={props.placeholder ? props.placeholder : "Password"}
				onChangeText={props.onChangeText}
				onBlur={props.onBlur}
				value={props.value}
				secureTextEntry={!showPassword}
			/>
			{props.value ?
				<Icon
					name={iconName}
					containerStyle={styles.icon}
					onPress={() => {
						setIconName(showPassword ? "visibility-off" : "visibility");
						setShowPassword(!showPassword);
					}}
				/>
				: null}

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
		flexDirection: "row",
		alignItems: "center",
	},
	input: {
		width: 300,
		padding: 10,
	},
	icon: {
		position: "absolute",
		right: 15,
	},
});
