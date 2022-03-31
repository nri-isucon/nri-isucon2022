package nriisucon.isubnb.response;

import java.time.LocalDate;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;
import nriisucon.isubnb.model.Activity;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class ReserveActivityDetail {
    private String reserveId;
    private LocalDate reserveDate;
    private int numberOfPeople;
    private Activity reserveActivity;
}
