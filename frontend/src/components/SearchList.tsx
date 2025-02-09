import { useNavigation } from "@react-navigation/native";
import React from "react";
import { SectionList, StyleSheet, Text } from "react-native";
import { Avatar, ListItem } from "react-native-elements";
import { Event, typeToString } from "../types/event";
import { User } from "../types/user";
import { ExploreNavigationProps } from "../utils/navigation";
import { Pressable } from "./Pressable";

interface Props {
	onEndReached?: () => void,
	users: User[] | undefined,
	events: Event[] | undefined
}

interface ItemProps {
	isUser?: boolean,
	id: string,
	image_url?: string,
	title: string,
	subtitle: string
}

export const SearchList = (props: Props) => {
	const navigation = useNavigation<ExploreNavigationProps>();

	const SectionListItem = (item: ItemProps) => (
		<Pressable
			onPress={() => {
				navigation.navigate(
					item.isUser ? "User" : "Event",
					item.isUser ? { user_id: item.id } : { event_id: item.id });
			}}
		>
			<ListItem bottomDivider>
				<Avatar
					rounded
					source={item.isUser && !item.image_url ? require("../../assets/icons/defUserAvatar.png") : { uri: item.image_url }}
				/>
				<ListItem.Content>
					<ListItem.Title>
						{item.title}
					</ListItem.Title>
					<ListItem.Subtitle>
						{item.subtitle}
					</ListItem.Subtitle>
				</ListItem.Content>
			</ListItem>
		</Pressable>
	);

	const sections = (): any => {
		return [
			{
				title: "User",
				data: props.users ? props.users : [],
				renderItem: ({ item }: { item: User }) => (
					<SectionListItem
						isUser
						id={item.id}
						title={item.name}
						subtitle={item.username}
						image_url={item.profile_image_url}
					/>
				),
			},
			{
				title: "Event",
				data: props.events ? props.events : [],
				renderItem: ({ item }: { item: Event }) => (
					<SectionListItem
						id={item.id}
						title={item.name}
						subtitle={"Type: " + typeToString(item.type)}
						image_url={item.logo_url}
					/>
				),
			},
		];
	};

	return <SectionList
		contentContainerStyle={styles.list}
		sections={sections()}
		keyExtractor={item => item.id}
		// @ts-ignore
		renderSectionHeader={({ section: { title, data } }) => data.length > 0 && <Text style={styles.header}>{title}</Text>}
		stickySectionHeadersEnabled
	/>;
};

const styles = StyleSheet.create({
	list: {
		paddingBottom: 90,
	},
	header: {
		fontSize: 16,
		fontWeight: "bold",
		backgroundColor: "#efefef",
		color: "black",
		paddingHorizontal: 15,
		paddingVertical: 15,
	},
});
