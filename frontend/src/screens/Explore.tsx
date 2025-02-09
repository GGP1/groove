import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { Region } from "react-native-maps";
import { API, Params } from "../api/api";
import { EventItem, ExploreHeader, List } from "../components";
import { FetchItemsResponse } from "../components/List";
import { SearchList } from "../components/SearchList";
import { EventsPag, UsersPag } from "../types/api";
import { Event } from "../types/event";
import { User } from "../types/user";
import { HTTP } from "../utils/http";
import { DEFAULT_REGION, getUserLocation } from "../utils/location";

export const App = () => {
	const [currentRegion, setCurrentRegion] = useState<Region>(DEFAULT_REGION);
	const [searching, setSearching] = useState(false);
	const [text, setText] = useState("");
	const [searchedEvents, setSearchedEvents] = useState<Event[]>();
	const [searchedUsers, setSearchedUsers] = useState<User[]>();

	useEffect(() => {
		getUserLocation(region => setCurrentRegion(region));
	}, []);

	const onChangeText = async (query: string) => {
		setText(query);
		setSearching(true);
		if (query.length === 0) {
			setSearching(false);
			return;
		} else if (query.length < 3) {
			setSearchedEvents(undefined);
			setSearchedUsers(undefined);
			return;
		}

		// TODO: Use cache when going backwards?
		// if (text.length > query.length) {}
		const results = await HTTP.paralell([
			API.Events.Search(query, { fields: ["id", "name", "type", "logo_url"] }),
			API.Users.Search(query, { fields: ["id", "name", "username", "profile_image_url"] }),
		]);

		const eventsResp = results[0] as EventsPag;
		const usersResp = results[1] as UsersPag;
		setSearchedEvents(eventsResp.events);
		setSearchedUsers(usersResp.users);
	};

	const getRecommendedEvents = async (params?: Params<Event>): FetchItemsResponse<Event> => {
		const resp = await API.Events.GetRecommended(currentRegion, {
			fields: ["id", "name", "type", "virtual", "cron", "start_date", "end_date", "slots", "ticket_type", "header_url"],
			...params,
		});
		return [resp.next_cursor, resp.events];
	};

	// TODO: on refresh we want to get different events (do not reset cursor)
	const EventsList = () => (
		<List<Event>
			fetchItems={(params) => getRecommendedEvents(params)}
			renderItem={({ item }) => <EventItem event={item} />}
			contentContainerStyle={styles.events}
		/>
	);

	return (
		<View>
			<ExploreHeader
				searchValue={text}
				onSearchChangeText={onChangeText}
				onSearchCancel={() => setSearching(false)}
			/>
			{searching ? <SearchList users={searchedUsers} events={searchedEvents} /> : <EventsList />}
		</View>
	);
};

const styles = StyleSheet.create({
	events: {
		paddingBottom: 100,
		width: "100%",
		alignItems: "center",
	},
});
