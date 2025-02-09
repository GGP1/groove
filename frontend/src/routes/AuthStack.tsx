import { createStackNavigator } from "@react-navigation/stack";
import React from "react";
import { GoBackIcon } from "../components";
import { Login, Register } from "../screens";

const { Navigator, Screen } = createStackNavigator();

export const AuthStack = () => {
	return (
		<Navigator initialRouteName="Login">
			<Screen name="Login" component={Login} options={{
				headerShown: false,
			}} />
			<Screen name="Register" component={Register} options={{
				headerTitle: "",
				headerTransparent: true,
				headerBackImage: () => GoBackIcon("black"),
			}} />
		</Navigator>
	);
};
