import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Permission } from "../../types/permission";

interface Props {
	permission: Permission
}

/** Permissionx is a component containing information about an event's permission. */
export const Permissionx = (props: Props) => {
	const { permission } = props;

	return (
		<View style={styles.container}>
			<Text>
				{permission.name},
				{permission.key},
				{permission.description},
				{permission.created_at}
			</Text>
		</View>
	);
};


const styles = StyleSheet.create({
	container: {

	},
});
