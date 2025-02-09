
import React from "react";
import { AuthProvider } from "../context/Session";
import Main from "./Main";

const Providers = () => {
	return (
		// Providers must wrap other components in a separate file (this one) to work.
		// If used in the same file the state is not updated.
		<AuthProvider>
			<Main />
		</AuthProvider>
	);
};

export default Providers;
