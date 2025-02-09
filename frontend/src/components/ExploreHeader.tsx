import { useNavigation } from "@react-navigation/native";
import React from "react";
import { StyleSheet } from "react-native";
import { Avatar, Header } from "react-native-elements";
import { SearchBar } from "../components";
import { ExploreNavigationProps } from "../utils/navigation";

interface Props {
	searchValue: string,
	onSearchChangeText: (text: string) => void,
	onSearchCancel: () => void;
}

export const ExploreHeader = (props: Props) => {
	const navigation = useNavigation<ExploreNavigationProps>();

	const mapIcon = () => (
		<Avatar
			source={require("../../assets/icons/map2.png")}
			size={36}
			onPress={() => navigation.navigate("Map")}
		/>
	);

	return (
		<Header
			backgroundColor="#0f1419"
			containerStyle={styles.container}
			leftComponent={mapIcon()}
			centerComponent={(
				<SearchBar
					value={props.searchValue}
					onChangeText={props.onSearchChangeText}
					onSearchCancel={props.onSearchCancel}
				/>
			)}
			leftContainerStyle={styles.sideContainer}
		/>
	);
};

const styles = StyleSheet.create({
	container: {
		backgroundColor: "white",
		borderBottomWidth: 0.9,
		borderBottomColor: "grey",
		height: 90,
	},
	sideContainer: {
		alignItems: "center",
	},
});
