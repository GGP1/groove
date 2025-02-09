import React from "react";
import { StyleSheet } from "react-native";
import { SearchBar as RNSearchBar } from "react-native-elements";

interface Props {
	value: string,
	onChangeText: (text: string) => void,
	onSearchCancel: () => void;
}

export const SearchBar = (props: Props) => {
	return <RNSearchBar
		placeholder="Search"
		round
		value={props.value}
		// @ts-ignore
		onChangeText={props.onChangeText}
		onCancel={props.onSearchCancel}
		autoCorrect={false}
		containerStyle={styles.container}
		inputContainerStyle={styles.inputContainer}
		inputStyle={styles.input}
		leftIconContainerStyle={styles.leftIcon}
	/>;
};


const styles = StyleSheet.create({
	container: {
		width: "98%",
		// height: 40,
		padding: 0,
		backgroundColor: "white",
		borderBottomColor: "transparent",
		borderTopColor: "transparent",
	},
	inputContainer: {
		height: 34,
		backgroundColor: "#f2f3f4",
	},
	input: {
		backgroundColor: "#f2f3f4",
		fontSize: 16,
		color: "black",
	},
	leftIcon: {
		paddingRight: 0,
	},
});
