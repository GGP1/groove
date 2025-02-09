import { useNavigation } from "@react-navigation/core";
import React, { useContext, useEffect, useState } from "react";
import { Dimensions, Linking, Platform, StyleSheet, Text, View } from "react-native";
import { Button, Icon } from "react-native-elements";
import FastImage from "react-native-fast-image";
import { List, PostItem } from "..";
import { API, Params } from "../../api/api";
import { SessionContext } from "../../context/Session";
import { Coordinates, Event, EventStatistics } from "../../types/event";
import { Post } from "../../types/post";
import { Role } from "../../types/role";
import { Cron } from "../../utils/cron";
import { formatDateNames } from "../../utils/date";
import { FetchItemsResponse } from "../List";
import { Pressable } from "../Pressable";

// Use sockets to update the availability of the tickets in real time to the users
// https://www.youtube.com/watch?v=cfggyE1Ptbc
// https://codedaily.io/tutorials/React-Native-and-Socketio

// Use react-native-tab-view for displaying multiple tabs horizontally, each one with a different content.

const screenDimensions = Dimensions.get("screen");

interface Props {
	id: string,
	item?: Event
}

/** Eventx is the main component in an event's profile. */
export const Eventx = (props: Props) => {
	const navigation = useNavigation();
	const { user } = useContext(SessionContext);
	const [event, setEvent] = useState<Event>();
	const [stats, setStats] = useState<EventStatistics>();
	const [duration, setDuration] = useState<number>(0);
	const [nextDate, setNextDate] = useState<Date>();
	const [authUserRole, setAuthUserRole] = useState<Role>();

	const getEventPosts = async (params?: Params<Post>): FetchItemsResponse<Post> => {
		const resp = await API.Events.GetPosts(props.id, params);
		return [resp.next_cursor, resp.posts];
	};

	useEffect(() => {
		const getEvent = async () => {
			if (props.item) {
				setEvent(props.item);
				return;
			}
			try {
				const e = await API.Events.GetByID(props.id);
				const [cronExpr, d] = Cron.parse(e.start_date, e.end_date, e.cron);
				setNextDate(cronExpr.next().toDate());
				setDuration(d);
				setEvent(e);
			} catch (err) {
				console.log(err);
			}
		};
		const getStats = async () => {
			try {
				const st = await API.Events.GetStatistics(props.id);
				setStats(st);
			} catch (err) {
				console.log(err);
			}
		};
		const getAuthUserRole = async () => {
			if (!user) {
				return;
			}

			try {
				const role = await API.Events.GetUserRole(props.id, { user_id: user?.id });
				setAuthUserRole(role);
			} catch (err) {
				console.log(err);
			}
		};
		const paralell = async () => {
			await Promise.all([getEvent(), getStats(), getAuthUserRole()]);
		};

		paralell();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [props, navigation, event?.name]);

	const Info = (p: { iconName: string, value: any, onPress?: () => void }) => {
		return (
			<Pressable style={styles.layDown} onPress={p.onPress}>
				<Icon name={p.iconName} containerStyle={styles.icon} size={30} />
				<Text style={styles.infoText}>{p.value}</Text>
			</Pressable>
		);
	};

	const geoURL = (l?: { address: string, coordinates: Coordinates }): string => {
		const scheme = Platform.select({ ios: "maps:0,0?q=", android: "geo:0,0?q=" });
		const latLng = `${l?.coordinates.latitude},${l?.coordinates.longitude}`;
		const label = l?.address;
		const url = Platform.select({
			ios: `${scheme}${label}@${latLng}`,
			android: `${scheme}${latLng}(${label})`,
		});
		return url || "";
	};

	const Header = () => (
		<View>
			<FastImage style={styles.header} source={{ uri: event?.header_url }} />
			<FastImage style={styles.logo} source={{ uri: event?.logo_url }} />

			<Text style={styles.title}>{event?.name}</Text>
			{/** TODO: include a button to display the description */}
			{event?.description && <Text style={styles.description}>{event?.description}</Text>}

			<View style={styles.infoContainer}>
				<View style={styles.layDown}>
					<View>
						<Info iconName="event" value={nextDate &&
							`${formatDateNames(nextDate)}\n-\n${formatDateNames(new Date(nextDate.getTime() + (duration * 1000 * 60)))}`
						} />
						{event?.url && <Info iconName="link" value={event?.url} onPress={async () => {
							if (event?.url) {
								await Linking.canOpenURL(event.url) && Linking.openURL(event.url);
							}
						}} />}
					</View>
					<View>
						<Info iconName="people" value={`${stats?.members_count}/${event?.slots} attendants`} />
						<Info iconName="place" value={event?.location.address} onPress={() => Linking.openURL(geoURL(event?.location))} />
					</View>
				</View>
			</View>

			{authUserRole ?
				<Button title="LEAVE" style={styles.button} onPress={() => API.Events.Leave(props.id)} />
				:
				<Button title="JOIN" style={styles.button} onPress={() => API.Events.Join(props.id)} />
			}
		</View>
	);

	return (
		<View style={styles.container}>
			<List<Post>
				ListHeaderComponent={Header}
				fetchItems={(params) => getEventPosts(params)}
				renderItem={({ item }) => <PostItem post={item} />}
			/>
		</View>
	);
};

// To: {nextDate && formatEventDate(new Date(nextDate.getTime() + (duration * 1000 * 60)))}
// const durationMin = duration % 60;
// Duration: {durationHs}hs {durationMin > 0 ? durationMin + "min" : ""}

const styles = StyleSheet.create({
	container: {
		flex: 1,
		backgroundColor: "white",
	},
	header: {
		width: screenDimensions.width,
		height: 160,
		resizeMode: "center",
	},
	logo: {
		position: "absolute",
		resizeMode: "center",
		borderWidth: 2,
		borderColor: "white",
		top: 100,
		left: 10,
		width: 90,
		height: 90,
		borderRadius: 50,
	},
	title: {
		fontSize: 26,
		textAlign: "center",
		marginVertical: 15,
	},
	description: {
		fontSize: 18,
		flexShrink: 1,
		marginBottom: 12,
		marginHorizontal: 20,
	},
	infoContainer: {
		marginBottom: 7,
	},
	button: {
		alignSelf: "flex-end",
		marginRight: 15,
		width: screenDimensions.width / 4,
		borderRadius: 50,
	},
	layDown: {
		flexDirection: "row",
		alignItems: "center",
		marginHorizontal: 7,
		flexWrap: "wrap",
	},
	icon: {
		marginRight: 5,
	},
	infoText: {
		fontSize: 16,
		textAlign: "center",
	},
});
