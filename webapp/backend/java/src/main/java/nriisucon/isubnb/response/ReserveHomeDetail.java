package nriisucon.isubnb.response;

import java.time.LocalDate;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;
import nriisucon.isubnb.model.Home;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class ReserveHomeDetail {
    private String reserveId;
    private LocalDate startDate;
    private LocalDate endDate;
    private int numberOfPeople;
    private Home reserveHome;
}
