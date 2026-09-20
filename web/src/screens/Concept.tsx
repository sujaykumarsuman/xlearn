import { useParams } from "react-router-dom";
import { Placeholder } from "../components/Placeholder";

export default function Concept() {
  const { slug } = useParams();
  return (
    <Placeholder
      eyebrow="Concept"
      title={slug ? slug.replace(/-/g, " ") : "Concept"}
      icon="bulb"
      sprint="S04"
      summary="Pattern reading with “when to reach for it” callouts and a Go code template."
    />
  );
}
