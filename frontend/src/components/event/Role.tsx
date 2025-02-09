import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Role } from "../../types/role";

interface Props {
	role: Role,
}

/** Rolex is the component used to display a user's role inside an event. */
export const Rolex = (props: Props) => {
	const {
		role,
	} = props;

	return (
		<View style={styles.container}>
			<Text>
				Name: {role.name},
				Permission keys: {role.permission_keys}
			</Text>
		</View>
	);
};


const styles = StyleSheet.create({
	container: {

	},
});
