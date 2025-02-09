import { useFocusEffect, useNavigation } from "@react-navigation/native";
import React, { useCallback, useContext, useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Button, Divider } from "react-native-elements";
import FastImage from "react-native-fast-image";
import { EventItem } from "..";
import { API, Params } from "../../api/api";
import { SessionContext } from "../../context/Session";
import { EventsPag, UsersPag } from "../../types/api";
import { Event } from "../../types/event";
import { Type, User, UserStatistics } from "../../types/user";
import { HTTP } from "../../utils/http";
import { MyProfileNavigationProps } from "../../utils/navigation";
import { FetchItemsResponse, List } from "../List";
import { Pressable } from "../Pressable";
import { UserItem } from "./UserItem";

interface Props {
	id: string,
	user?: User
}

// TODO: user logo image should have a border with the same color as the theme

export const Userx = (props: Props) => {
	const { user: authUser } = useContext(SessionContext);
	const [user, setUser] = useState<User>();
	const [stats, setStats] = useState<UserStatistics>();
	// areRelated holds a value that determines if the auth user is related to the visited one
	const [areRelated, setAreRelated] = useState<boolean>();
	const [isPersonalType, setIsPersonalType] = useState<boolean>();
	const isAuthUser = props.id === authUser?.id || props.user?.id === authUser?.id;
	const navigation = useNavigation<MyProfileNavigationProps>();

	useFocusEffect(useCallback(() => {
		const areFriends = async (u: User, personal: boolean) => {
			if (isAuthUser || !authUser) {
				return;
			}
			try {
				if (personal) {
					const resp = await API.Users.GetFriends<UsersPag>(authUser.id, { lookup_id: u.id });
					setAreRelated(resp.users && resp.users.length === 1);
				} else {
					const resp = await API.Users.GetFollowing<UsersPag>(authUser.id, { lookup_id: u.id });
					setAreRelated(resp.users && resp.users.length === 1);
				}
			} catch (err) {
				console.log(err);
			}
		};
		const setState = async (u: User) => {
			setUser(u);
			navigation.setOptions({
				headerTitle: u.username,
			});
			const personal = u.type === Type.Personal;
			setIsPersonalType(personal);
			await areFriends(u, personal);
		};
		const getUser = async () => {
			if (props.user) {
				await setState(props.user);
				return;
			}

			try {
				const u = await API.Users.GetByID(props.id);
				await setState(u);
			} catch (err) {
				console.log(err);
			}
		};
		const getStats = async () => {
			try {
				const st = await API.Users.GetStatistics(props.id);
				setStats(st);
			} catch (err) {
				console.log(err);
			}
		};
		const paralell = async () => {
			await HTTP.paralell([getUser(), getStats()]);
		};

		paralell();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []));

	const FieldValue = (p: { field: string, value?: number, onPress: () => void }) => (
		<Pressable style={styles.fieldsContainer} onPress={p.onPress}>
			<Text style={styles.value}>
				{p.value}
			</Text>
			<Text style={styles.field}>
				{p.field}
			</Text>
		</Pressable>
	);

	const EventList = (p: {
		eventType: string,
		fetchItems: (params?: Params<Event>) => FetchItemsResponse<Event>,
		count?: number
	}) => {
		if (!p.count || p.count <= 0) {
			return null;
		}
		return (
			<View>
				<Text style={styles.text}>{p.eventType} events · {p.count}</Text>
				<List<Event>
					renderItem={({ item }) => <EventItem event={item} />}
					fetchItems={(params) => p.fetchItems(params)}
					horizontal
					showsHorizontalScrollIndicator={false}
				/>
			</View>
		);
	};

	return (
		<View style={styles.container}>

			<View style={styles.header}>
				<FastImage
					source={user?.profile_image_url ? { uri: user.profile_image_url } : require("../../../assets/icons/defUserAvatar.png")}
					style={styles.image}
				/>
				<View style={styles.infoContainer}>
					<Text style={styles.name}>
						{user?.name}
					</Text>
					<View style={styles.fieldsContainer}>
						{isPersonalType ?
							<FieldValue
								field={"friends"}
								value={stats?.friends_count}
								onPress={() => navigation.push("List", {
									title: "Friends",
									props: {
										fetchItems: (params) => getFriends(props.id, params),
										renderItem: ({ item }) => <UserItem user={item} />,
									},
								})}
							/>
							:
							<FieldValue
								field={"followers"}
								value={stats?.followers_count}
								onPress={() => navigation.push("List", {
									title: "Followers",
									props: {
										fetchItems: (params) => getFollowers(props.id, params),
										renderItem: ({ item }) => <UserItem user={item} />,
									},
								})}
							/>
						}
						<FieldValue
							field={"following"}
							value={stats?.following_count}
							onPress={() => navigation.push("List", {
								title: "Following",
								props: {
									fetchItems: (params) => getFollowing(props.id, params),
									renderItem: ({ item }) => <UserItem user={item} />,
								},
							})}
						/>
					</View>

					{isAuthUser ? (
						<Button
							title="EDIT PROFILE"
							buttonStyle={styles.editButton}
							onPress={() => user && navigation.navigate("UpdateUser", { user: user })}
						/>
					) : areRelated ? (
						<Button
							title={isPersonalType ? "REMOVE FRIEND" : "UNFOLLOW"}
							buttonStyle={styles.removeButton}
							onPress={() => {
								if (user && authUser) {
									isPersonalType
										? API.Users.RemoveFriend(authUser.id, { friend_id: user.id })
										: API.Users.Unfollow(authUser.id, user.id);
								}
							}}
						/>
					) : (
						<Button
							title={isPersonalType ? "ADD FRIEND" : "FOLLOW"}
							buttonStyle={styles.addButton}
							onPress={() => {
								if (user && authUser) {
									isPersonalType
										? API.Users.SendFriendRequest(authUser.id, { user_id: user.id })
										: API.Users.Follow(authUser.id, user.id);
								}
							}}
						/>
					)}
				</View>
			</View>

			{/* TODO: if the description contains URLs or phone numbers, create hyperlinks */}
			{user?.description && <Text style={styles.description}>{user?.description}</Text>}

			<Divider style={styles.divider} />

			<View style={styles.lists}>
				<EventList
					eventType="Hosted"
					fetchItems={(params) => getHostedEvents(props.id, params)}
					count={stats?.hosted_events_count}
				/>
				<EventList
					eventType="Attending"
					fetchItems={(params) => getAttendingEvents(props.id, params)}
					count={stats?.attending_events_count}
				/>
			</View>
		</View>
	);
};

const getFriends = async (id: string, params?: Params<User>): FetchItemsResponse<User> => {
	const resp = await API.Users.GetFriends<UsersPag>(id, {
		fields: ["id", "username", "profile_image_url", "name"],
		...params,
	});
	return [resp.next_cursor, resp.users];
};
const getFollowers = async (id: string, params?: Params<User>): FetchItemsResponse<User> => {
	const resp = await API.Users.GetFollowers<UsersPag>(id, {
		fields: ["id", "username", "profile_image_url", "name"],
		...params,
	});
	return [resp.next_cursor, resp.users];
};
const getFollowing = async (id: string, params?: Params<User>): FetchItemsResponse<User> => {
	const resp = await API.Users.GetFollowing<UsersPag>(id, {
		fields: ["id", "username", "profile_image_url", "name"],
		...params,
	});
	return [resp.next_cursor, resp.users];
};
const getHostedEvents = async (id: string, params?: Params<Event>): FetchItemsResponse<Event> => {
	const resp = await API.Users.GetHostedEvents<EventsPag>(id, {
		fields: ["id", "name", "type", "virtual", "cron", "start_date", "end_date", "slots", "ticket_type", "header_url"],
		...params,
	});
	return [resp.next_cursor, resp.events];
};
const getAttendingEvents = async (id: string, params?: Params<Event>): FetchItemsResponse<Event> => {
	const resp = await API.Users.GetAttendingEvents<EventsPag>(id, {
		fields: ["id", "name", "type", "virtual", "cron", "start_date", "end_date", "slots", "ticket_type", "header_url"],
		...params,
	});
	return [resp.next_cursor, resp.events];
};

const styles = StyleSheet.create({
	container: {
		marginTop: 20,
	},
	header: {
		marginHorizontal: 20,
		marginBottom: 10,
		marginTop: -5,
		flexDirection: "row",
		alignItems: "center",
	},
	infoContainer: {
		marginLeft: 25,
		width: 190,
	},
	image: {
		width: 120,
		height: 120,
		borderRadius: 100,
		borderWidth: 0.5,
		borderColor: "rgba(0,0,0,0.2)",
	},
	name: {
		fontSize: 20,
	},
	description: {
		fontSize: 16,
		marginHorizontal: 20,
	},
	fieldsContainer: {
		flexDirection: "row",
		marginVertical: 5,
		justifyContent: "space-evenly",
	},
	value: {
		fontWeight: "bold",
		paddingRight: 3,
		fontSize: 18,
	},
	field: {
		color: "rgba(0,0,0,0.75)",
		marginRight: 10,
		fontSize: 18,
	},
	editButton: {
		backgroundColor: "rgba(255, 0, 0, 0.5)",
		padding: 6,
	},
	addButton: {
		padding: 6,
	},
	removeButton: {
		padding: 6,
		backgroundColor: "rgba(255, 0, 0, 1)",
	},
	divider: {
		marginTop: 5,
	},
	lists: {
		margin: 15,
	},
	text: {
		fontSize: 19,
		textAlign: "center",
		fontWeight: "bold",
	},
});
