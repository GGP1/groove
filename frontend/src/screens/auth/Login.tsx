import React, { useContext, useState } from "react";
import { Keyboard, StyleSheet, Text, TouchableWithoutFeedback, View } from "react-native";
import { Button } from "react-native-elements";
import { Input, PasswordInput } from "../../components";
import { SessionContext } from "../../context/Session";
import { AuthNavProps } from "../../utils/navigation";

export const App = ({ navigation }: AuthNavProps<"Login">) => {
	const { login } = useContext(SessionContext);
	const [username, setUsername] = useState<string>();
	const [password, setPassword] = useState<string>();

	return (
		<TouchableWithoutFeedback onPress={Keyboard.dismiss} style={styles.touchable}>
			<View style={styles.touchable}>
				<Text style={styles.title}>Login</Text>
				<Input
					placeholder="Username"
					onChangeText={(text) => setUsername(text)}
					autoCapitalize="none"
					autoCompleteType="username"
				/>
				<PasswordInput
					onChangeText={(text) => setPassword(text)}
					value={password}
				/>
				<Button title="Login" onPress={() => login({ username, password })} />
				<Text
					style={styles.registerLink}
					onPress={() => navigation.navigate("Register")}
				>
					Do not have an account? Register
				</Text>
			</View>
		</TouchableWithoutFeedback>
	);
};

const styles = StyleSheet.create({
	registerLink: {
		color: "blue",
		textDecorationLine: "underline",
	},
	touchable: {
		flex: 1,
		alignItems: "center",
		justifyContent: "center",
	},
	title: {
		fontSize: 17,
		marginBottom: 15,
	},
});
