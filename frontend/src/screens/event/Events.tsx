import React, { useContext, useEffect, useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Agenda } from "react-native-calendars";
import { FAB } from "react-native-elements";
import { SafeAreaView } from "react-native-safe-area-context";
import { API } from "../../api/api";
import { Center } from "../../components";
import { SessionContext } from "../../context/Session";
import { EventsPag } from "../../types/api";
import { Event } from "../../types/event";
import { EventsNavProps } from "../../utils/navigation";

// This tab could include a calendar/agenda
// A "history" button showing past events

export const App = ({ navigation }: EventsNavProps<"Events">) => {
	const { user } = useContext(SessionContext);
	const [events, setEvents] = useState<Event[] | null>(null);
	const [calendarToggled, setCalendarToggled] = useState<boolean>(false);

	useEffect(() => {
		const getEvents = async () => {
			if (user) {
				try {
					const resp = await API.Users.GetAttendingEvents<EventsPag>(user.id);
					setEvents(resp.events);
				} catch (err) {
					console.log(err);
				}
			}
		};
		getEvents();

	}, [user]);

	const renderEvent = (event: Event) => (
		<View style={styles.item}>
			<Text>{event.name}</Text>
		</View>
	);

	const renderEmpty = () => (
		<Center>
			<Text style={styles.emptyMsg}>No events</Text>
		</Center>
	);

	return (
		<SafeAreaView style={styles.container}>
			{/*
			<List<Event>
				fetchItems={(params) => getEvents(params)}
				renderItem={({ item }) => (
					<Pressable onPress={() => {
						navigation.navigate("Event", { eventID: event.id, item: event });
					}}>
						<View style={styles.event}>
							<Text>{event.id}</Text>
							<Text>{event.slots}</Text>
						</View>
					</Pressable>
				)}
			/>
			*/}
			<Agenda
				items={{
					"2022-01-16": [{ name: "Tomorrowland", day: "", height: 0 }],
				} || events}
				// @ts-ignore
				renderItem={(event) => renderEvent(event)}
				renderEmptyData={() => renderEmpty()}
				minDate={new Date("2022-01-02").toDateString()}
				maxDate={new Date("2100-10-10").toDateString()}
				pastScrollRange={120}
				futureScrollRange={36}
				hideKnob={false}
				onCalendarToggled={(toggled) => setCalendarToggled(toggled)}
				showClosingKnob
				showOnlySelectedDayItems
				theme={{
					dotColor: "coral",
					agendaKnobColor: "rgba(220, 20, 60, 0.7)",
					agendaTodayColor: "red",
					selectedDotColor: "crimson",
					selectedDayBackgroundColor: "crimson",
					todayTextColor: "red",
				}}
			/>

			<FAB
				visible={!calendarToggled}
				placement={"right"}
				title="New"
				color={"crimson"} // #F02A4B
				onPress={() => navigation.push("CreateEvent")}
			/>
		</SafeAreaView>
	);
};

/*
<View>
	<Text style={styles.dayNum}>
		16
	</Text>
	<Text style={styles.dayText}>
		Sun
	</Text>
</View>
dayNum: {
		fontSize: 28,
		fontWeight: "200",
		fontFamily: "System",
		color: "#7a92a5",
	},
	dayText: {
		fontSize: 14,
		fontWeight: "300",
		fontFamily: "System",
		color: "#7a92a5",
		backgroundColor: "rgba(0,0,0,0)",
		marginTop: -5,
	},
*/

const styles = StyleSheet.create({
	container: {
		flex: 1,
	},
	event: {
		flex: 1,
		margin: 15,
		padding: 10,
		borderRadius: 10,
		borderWidth: 1.4,
	},
	item: {
		backgroundColor: "white",
		flex: 1,
		height: 50,
		borderRadius: 5,
		padding: 10,
		marginRight: 10,
		marginTop: 17,
	},
	emptyMsg: {
		bottom: 22,
		fontSize: 17,
	},
});
