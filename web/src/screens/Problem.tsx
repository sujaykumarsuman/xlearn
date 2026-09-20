import { useParams } from "react-router-dom";
import { Placeholder } from "../components/Placeholder";

export default function Problem() {
  const { id } = useParams();
  return (
    <Placeholder
      eyebrow="Guided problem"
      title={id ? `Problem #${id}` : "Problem"}
      icon="code"
      sprint="S05"
      summary="The guided 3-pane workspace: statement · editor · timer HUD, with gated hint/solution reveal and outcome logging."
    />
  );
}
