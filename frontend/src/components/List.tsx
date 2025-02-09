import { useFocusEffect } from "@react-navigation/native";
import React, { useCallback, useRef, useState } from "react";
import { FlatList, FlatListProps } from "react-native";
import { Params } from "../api/api";

interface ID {
	id: string
}

interface Name {
	name: string
}

type FLProps<T> = Omit<
	FlatListProps<T>,
	"data" | "keyExtractor" | "onEndReached" | "onRefresh" | "refreshing" | "onMomentumScrollBegin"
>

export type FetchItemsResponse<T> = Promise<[string, T[] | undefined] | undefined>

export interface Props<T> extends FLProps<T> {
	fetchItems: (params?: Params<T>) => FetchItemsResponse<T>
}

/** List component implements a FlatList of all types with an id field.
 * It handles pagination and refreshing by itself.
 *
 * Usage: <List\<T> ... />
 *
 * To add params in the List "fetchItems": {fields: ["id"], ...params}
 */
export const List = <T extends ID | Name>(props: Props<T>) => {
	const [items, setItems] = useState<readonly T[]>();
	const [cursor, setCursor] = useState<string>();
	const [refreshing, setRefreshing] = useState<boolean>(false);
	const isMounted = useRef(false);

	const getItems = useCallback(async () => {
		try {
			const resp = await props.fetchItems({ cursor: cursor });
			if (!resp || !isMounted.current) {
				return;
			}
			const [nextCursor, elems] = resp;
			if (!elems) {
				return;
			}
			setCursor(nextCursor);
			// If we already hold items, append the new ones to them
			items ? setItems(items.concat(elems)) : setItems(elems);
		} catch (err) {
			console.log("List", err);
		} finally {
			setRefreshing(false);
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	// TODO: do we want to re-render a list everytime it's focused?
	useFocusEffect(useCallback(() => {
		isMounted.current = true;
		getItems();

		return () => { isMounted.current = false; };
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []));

	return (
		<FlatList
			data={items}
			keyExtractor={item => instanceOfID(item) ? item.id : item.name}
			onEndReachedThreshold={props.onEndReachedThreshold ? props.onEndReachedThreshold : 0.1}
			onEndReached={({ distanceFromEnd }) => distanceFromEnd < 0 ? undefined : getItems()}
			refreshing={refreshing}
			onRefresh={() => {
				setRefreshing(true);
				setCursor(undefined);
				setItems(undefined);
				getItems();
			}}
			{...props}
		/>
	);
};

/** instanceOfID takes an item and returns if it contains an "id" field or not */
function instanceOfID(item: ID | Name): item is ID {
	return "id" in item;
}
