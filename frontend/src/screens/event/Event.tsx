import React from "react";
import { Eventx } from "../../components";
import { CommonNavProps } from "../../utils/navigation";

export const App = ({ route }: CommonNavProps<"Event">) => {
	return (
		<Eventx id={route.params.event_id} item={route.params.item} />
	);
};
