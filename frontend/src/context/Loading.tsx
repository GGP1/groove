import React, { useState } from "react";

type ContextProps = {
	loading: boolean,
	setLoading: (value: boolean) => void,
}

export const LoadingContext = React.createContext<ContextProps>({
	loading: false,
	setLoading: () => { },
});

interface Props {
	children: React.ReactNode;
}

export const LoadingProvider = (props: Props) => {
	const [loading, setLoading] = useState<boolean>(false);

	return (
		<LoadingContext.Provider
			value={{
				loading: loading,
				setLoading: (value) => setLoading(value),
			}}
		>
			{props.children}
		</LoadingContext.Provider>
	);
};
