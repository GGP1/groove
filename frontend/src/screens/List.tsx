import { useNavigation } from "@react-navigation/core";
import React, { useEffect } from "react";
import { LogBox, View } from "react-native";
import { List } from "../components";
import { HomeNavProps } from "../utils/navigation";

export const App = ({ route }: HomeNavProps<"List">) => {
	const navigation = useNavigation();

	LogBox.ignoreLogs([
		"Non-serializable values were found in the navigation state",
	]);

	useEffect(() => {
		// TODO: when loading two lists in the same stack it only shows the first one
		//  StackActions.replace("List", route.params.props); // not working
		navigation.setOptions({
			headerTitle: route.params.title,
		});
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	return (
		<View>
			<List {...route.params.props} />
		</View>
	);
};
