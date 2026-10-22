import { Heading, Shell, Text } from "./parts";

export default function UnknownAddress({ unreachable = false }: { unreachable?: boolean }) {
  return (
    <Shell>
      <div className="pe-login">
        {unreachable ? (
          <>
            <Heading title="PhoenixEvent is not answering" />
            <Text>Try again in a moment.</Text>
          </>
        ) : (
          <>
            <Heading title="No organization here" />
            <Text>There is no organization at this address. Check the address with your organization.</Text>
          </>
        )}
      </div>
    </Shell>
  );
}
